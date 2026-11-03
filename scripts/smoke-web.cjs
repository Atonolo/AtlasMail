'use strict';

const path = require('node:path');
const crypto = require('node:crypto');
const { app, BrowserWindow } = require('electron');
const { StateStore } = require('../src/core/store.cjs');
const { MailService } = require('../src/core/service.cjs');
const { createServer } = require('../src/server/server.cjs');
const keyboard = require('./smoke-keyboard.cjs');

const directory = path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'data');
const token = crypto.randomBytes(32).toString('base64url');
const store = new StateStore(directory);
const server = createServer({ service: new MailService(store), token });
app.setPath('userData', path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'profile'));

async function exercise(window) {
  await keyboard.exerciseLanguage(window);
  await keyboard.connect(window, token);
  await keyboard.waitFor(window, "!document.getElementById('dashboard').hidden", 'Timed out connecting to the web shell');
  await keyboard.addAccount(window, 'Web smoke', 'web@example.invalid');
  const result = await window.webContents.executeJavaScript(`(async () => {
    const wait = async (condition) => {
      for (let n = 0; n < 100; n++) {
        if (condition()) return;
        await new Promise(resolve => setTimeout(resolve, 50));
      }
      throw new Error('Timed out waiting for the web shell');
    };
    await wait(() => document.getElementById('account-count').textContent === '1');
    const saved = document.getElementById('account-list').textContent.includes('web@example.invalid')
      && !JSON.stringify(localStorage).includes(${JSON.stringify(token)});
    const originalFetch = window.fetch;
    let release;
    let requestFinished = false;
    window.fetch = (url, options) => new Promise((resolve, reject) => {
      release = () => originalFetch(url, options).then((response) => {
        requestFinished = true;
        resolve(response);
      }, reject);
    });
    try {
      document.getElementById('retry').click();
      await wait(() => Boolean(release));
      document.getElementById('disconnect').click();
      release();
      await wait(() => requestFinished);
      await new Promise(resolve => setTimeout(resolve, 50));
      return saved && !document.getElementById('login-section').hidden
        && document.getElementById('dashboard').hidden;
    } finally {
      window.fetch = originalFetch;
    }
  })()`);
  if (!result) throw new Error('Web account, token handling or disconnect isolation failed');
  const reloaded = new StateStore(directory).read();
  if (reloaded.accounts.length !== 1 || reloaded.accounts[0].address !== 'web@example.invalid') {
    throw new Error('Web account was not persisted');
  }
}

app.whenReady().then(async () => {
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const window = new BrowserWindow({
    show: false,
    webPreferences: { sandbox: true, contextIsolation: true, nodeIntegration: false },
  });
  const readyToShow = new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Timed out waiting for the web window to render')), 10_000);
    window.once('ready-to-show', () => { clearTimeout(timer); resolve(); });
  });
  await Promise.all([window.loadURL(`http://127.0.0.1:${server.address().port}/`), readyToShow]);
  await exercise(window);
  console.log('Web keyboard shell, authentication and persistence smoke passed.');
  await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  app.exit(0);
}).catch((error) => {
  console.error(`Web smoke failed: ${error.message}`);
  server.close(() => app.exit(1));
  if (!server.listening) app.exit(1);
});
