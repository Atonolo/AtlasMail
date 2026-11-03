'use strict';

const path = require('node:path');
const http = require('node:http');
const { app, BrowserWindow } = require('electron');
const { StateStore } = require('../src/core/store.cjs');
const keyboard = require('./smoke-keyboard.cjs');

const directory = path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'data');
process.env.ATLAS_MAIL_DATA_DIR = directory;
app.setPath('userData', path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'profile'));
require('../src/desktop/main.cjs');

async function exercise(window) {
  if (window.webContents.isLoading()) {
    await new Promise((resolve, reject) => {
      window.webContents.once('did-finish-load', resolve);
      window.webContents.once('did-fail-load', (_event, code) => reject(new Error(`Page load failed: ${code}`)));
    });
  }
  await keyboard.waitFor(window, "!document.getElementById('dashboard').hidden", 'Timed out waiting for the desktop shell');
  await keyboard.exerciseLanguage(window);
  await keyboard.addAccount(window, 'Desktop smoke', 'desktop@example.invalid');
  await keyboard.waitFor(window, "document.getElementById('account-count').textContent === '1'", 'Timed out waiting for keyboard account submission');
  const result = await window.webContents.executeJavaScript("document.getElementById('account-list').textContent.includes('desktop@example.invalid')");
  if (!result) throw new Error('Desktop account was not displayed');
  const reloaded = new StateStore(directory).read();
  if (reloaded.accounts.length !== 1 || reloaded.accounts[0].address !== 'desktop@example.invalid') {
    throw new Error('Desktop account was not persisted');
  }

  let requests = 0;
  const server = http.createServer((_request, response) => {
    requests += 1;
    response.end('unexpected');
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  try {
    let blocked = false;
    try {
      await window.webContents.session.fetch(`http://127.0.0.1:${server.address().port}/`);
    } catch (error) {
      if (!String(error.message).includes('ERR_BLOCKED_BY_CLIENT')) throw error;
      blocked = true;
    }
    if (!blocked || requests !== 0) throw new Error('Desktop network request was not blocked');
  } finally {
    await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
}

app.whenReady().then(async () => {
  const window = BrowserWindow.getAllWindows()[0];
  if (!window) throw new Error('Desktop window was not created');
  await exercise(window);
  console.log('Desktop keyboard shell, persistence and network isolation smoke passed.');
  app.exit(0);
}).catch((error) => {
  console.error(`Desktop smoke failed: ${error.message}`);
  app.exit(1);
});
