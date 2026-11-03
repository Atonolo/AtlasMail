'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawn } = require('node:child_process');
const { StateStore } = require('../src/core/store.cjs');
const { MailService } = require('../src/core/service.cjs');
const { createServer } = require('../src/server/server.cjs');
const {
  firefoxArguments, firefoxReportMatches, firefoxReportPath, firefoxRuntimeIdentity,
} = require('./firefox-runtime.cjs');

const timeoutMs = 20_000;

function firefoxEndpoint(child) {
  return new Promise((resolve, reject) => {
    let output = '';
    let finished = false;
    const timer = setTimeout(() => finish(new Error(`Firefox did not open WebDriver BiDi: ${output}`)), timeoutMs);
    function finish(error, endpoint) {
      if (finished) return;
      finished = true;
      clearTimeout(timer);
      child.removeListener('error', onError);
      child.removeListener('exit', onExit);
      child.stdout.removeListener('data', onData);
      child.stderr.removeListener('data', onData);
      child.stdout.resume();
      child.stderr.resume();
      if (error) reject(error);
      else resolve(endpoint);
    }
    function onError(error) { finish(error); }
    function onExit(code) { finish(new Error(`Firefox exited before WebDriver BiDi started (${code}): ${output}`)); }
    function onData(chunk) {
      output = (output + chunk.toString()).slice(-4000);
      const match = output.match(/WebDriver BiDi listening on (ws:\/\/127\.0\.0\.1:\d+)/);
      if (match) finish(null, `${match[1]}/session`);
    }
    child.stdout.on('data', onData);
    child.stderr.on('data', onData);
    child.once('error', onError);
    child.once('exit', onExit);
  });
}

async function connect(endpoint) {
  const socket = new WebSocket(endpoint);
  await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Firefox WebDriver connection timed out')), timeoutMs);
    socket.addEventListener('open', () => { clearTimeout(timer); resolve(); }, { once: true });
    socket.addEventListener('error', () => {
      clearTimeout(timer);
      reject(new Error('Firefox WebDriver connection failed'));
    }, { once: true });
  });

  let nextId = 0;
  const pending = new Map();
  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data);
    const request = pending.get(message.id);
    if (!request) return;
    pending.delete(message.id);
    clearTimeout(request.timer);
    if (message.type === 'error') request.reject(new Error(`${request.method}: ${message.error}: ${message.message}`));
    else request.resolve(message.result);
  });
  socket.addEventListener('close', () => {
    for (const request of pending.values()) {
      clearTimeout(request.timer);
      request.reject(new Error('Firefox WebDriver connection closed'));
    }
    pending.clear();
  });

  return {
    close: () => socket.close(),
    command(method, params = {}) {
      return new Promise((resolve, reject) => {
        const id = ++nextId;
        const timer = setTimeout(() => {
          pending.delete(id);
          reject(new Error(`Firefox WebDriver command timed out: ${method}`));
        }, timeoutMs);
        pending.set(id, { method, resolve, reject, timer });
        socket.send(JSON.stringify({ id, method, params }));
      });
    },
  };
}

async function main() {
  const reportPath = firefoxReportPath();
  fs.rmSync(reportPath, { force: true });
  const runtime = firefoxRuntimeIdentity();
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-firefox-smoke-'));
  const profile = path.join(directory, 'profile');
  fs.mkdirSync(profile);
  const store = new StateStore(path.join(directory, 'data'));
  const token = crypto.randomBytes(32).toString('base64url');
  const server = createServer({ service: new MailService(store), token });
  let child;
  let exited;
  let browser;
  let failure;
  let successMessage;
  let browserVersion;
  try {
    await new Promise((resolve, reject) => {
      server.once('error', reject);
      server.listen(0, '127.0.0.1', resolve);
    });
    const origin = `http://127.0.0.1:${server.address().port}`;
    child = spawn(runtime.executable, firefoxArguments(profile), {
      env: { ...process.env, MOZ_NO_REMOTE: '1' }, stdio: ['ignore', 'pipe', 'pipe'],
    });
    exited = new Promise((resolve) => child.once('close', resolve));
    const endpoint = await firefoxEndpoint(child);
    browser = await connect(endpoint);
    const session = await browser.command('session.new', { capabilities: {} });
    browserVersion = session.capabilities.browserVersion;
    if (typeof browserVersion !== 'string' || !browserVersion.trim()) {
      throw new Error('Firefox did not report its browser version');
    }
    const tree = await browser.command('browsingContext.getTree');
    if (!tree.contexts.length) throw new Error('Firefox opened no browsing context');
    const context = tree.contexts[0].context;
    await browser.command('browsingContext.navigate', { context, url: `${origin}/`, wait: 'complete' });
    await browser.command('browsingContext.activate', { context });

    async function evaluate(expression) {
      const result = await browser.command('script.evaluate', { expression, target: { context }, awaitPromise: true });
      if (result.type !== 'success') throw new Error(`Firefox page script failed: ${result.exceptionDetails.text}`);
      return result.result.value;
    }
    async function waitFor(expression, message) {
      for (let attempt = 0; attempt < 100; attempt += 1) {
        if (await evaluate(expression)) return;
        await new Promise((resolve) => setTimeout(resolve, 50));
      }
      throw new Error(message);
    }
    async function click(selector) {
      const position = JSON.parse(await evaluate(`(() => {
        const element = document.querySelector(${JSON.stringify(selector)});
        element.scrollIntoView({ block: 'center', inline: 'center' });
        const bounds = element.getBoundingClientRect();
        return JSON.stringify({ x: Math.round(bounds.left + bounds.width / 2), y: Math.round(bounds.top + bounds.height / 2) });
      })()`));
      await browser.command('input.performActions', {
        context,
        actions: [{ type: 'pointer', id: 'mouse', parameters: { pointerType: 'mouse' }, actions: [
          { type: 'pointerMove', ...position, origin: 'viewport' },
          { type: 'pointerDown', button: 0 }, { type: 'pointerUp', button: 0 },
        ] }],
      });
    }
    async function fill(selector, value) {
      await evaluate(`(() => {
        const element = document.querySelector(${JSON.stringify(selector)});
        element.value = ${JSON.stringify(value)};
        element.dispatchEvent(new Event('input', { bubbles: true }));
      })()`);
    }
    await click('#language-en');
    await waitFor("document.documentElement.lang === 'en'", 'English language selection failed');
    await click('#language-de');
    await waitFor("document.documentElement.lang === 'de'", 'German language selection failed');

    async function connectToService() {
      await fill('#token', token);
      await click('#login-form button[type=submit]');
      await waitFor("!document.getElementById('dashboard').hidden", 'Firefox could not connect to the web service');
    }
    await connectToService();
    await fill('#account-name', 'Firefox smoke');
    await fill('#account-address', 'firefox@example.invalid');
    await click('#save-account');
    await waitFor("document.getElementById('account-list').textContent.includes('firefox@example.invalid')",
      'Firefox could not save the synthetic account');
    const savedAccounts = store.read().accounts;
    if (savedAccounts.length !== 1 || savedAccounts[0].address !== 'firefox@example.invalid') {
      throw new Error('Firefox account was not persisted');
    }
    if (await evaluate(`JSON.stringify(localStorage).includes(${JSON.stringify(token)}) ||
      JSON.stringify(sessionStorage).includes(${JSON.stringify(token)}) || document.getElementById('token').value !== ''`)) {
      throw new Error('Firefox retained the access token in browser storage or the login field');
    }
    await browser.command('browsingContext.reload', { context, wait: 'complete' });
    await waitFor("!document.getElementById('login-section').hidden", 'Reload did not require a new login');
    await connectToService();
    await waitFor("document.getElementById('account-list').textContent.includes('firefox@example.invalid')",
      'Firefox did not reload the saved account');
    const resources = JSON.parse(await evaluate("JSON.stringify(performance.getEntriesByType('resource').map(entry => entry.name))"));
    if (!resources.length || resources.some((url) => new URL(url).origin !== origin)) {
      throw new Error('Firefox page loaded an unexpected resource origin');
    }
    successMessage = `Firefox ${browserVersion} (headless): web authentication, persistence and reload smoke passed.`;
  } catch (error) {
    failure = error;
  } finally {
    const cleanupErrors = [];
    if (browser) {
      try {
        await browser.command('browser.close');
      } catch (error) {
        cleanupErrors.push(error);
      } finally {
        browser.close();
      }
    }
    if (child) {
      if ((cleanupErrors.length || !browser) && child.exitCode === null && child.signalCode === null) child.kill();
      let timer;
      const stopped = await Promise.race([
        exited.then(() => true),
        new Promise((resolve) => { timer = setTimeout(() => resolve(false), 10_000); }),
      ]);
      clearTimeout(timer);
      if (!stopped) {
        child.kill();
        child.stdout.destroy();
        child.stderr.destroy();
        cleanupErrors.push(new Error('Firefox did not exit after the smoke test'));
      }
    }
    try {
      if (server.listening) await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
    } catch (error) {
      cleanupErrors.push(error);
    }
    try {
      fs.rmSync(directory, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 });
    } catch (error) {
      cleanupErrors.push(error);
    }
    if (cleanupErrors.length) {
      const details = cleanupErrors.map((error) => error.message).join('; ');
      if (failure) throw new Error(`${failure.message}; cleanup failed: ${details}`);
      throw new Error(`Firefox smoke cleanup failed: ${details}`);
    }
  }
  if (failure) throw failure;
  const report = { schemaVersion: 2, browserVersion, ...runtime };
  if (!firefoxReportMatches(report, firefoxRuntimeIdentity())) {
    throw new Error('Firefox runtime changed during the smoke test');
  }
  fs.mkdirSync(path.dirname(reportPath), { recursive: true });
  fs.writeFileSync(reportPath, `${JSON.stringify(report, null, 2)}\n`, { flag: 'wx' });
  console.log(successMessage);
}

main().catch((error) => {
  console.error(`Firefox smoke failed: ${error.message}`);
  process.exitCode = 1;
});
