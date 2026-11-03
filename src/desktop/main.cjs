'use strict';

const path = require('node:path');
const { app, BrowserWindow, dialog, ipcMain, safeStorage, session } = require('electron');
const { ModelError } = require('../core/model.cjs');
const { StateStore, StoreError } = require('../core/store.cjs');
const { dataDirectory } = require('../core/paths.cjs');
const { MailService } = require('../core/service.cjs');
const { CredentialStore } = require('./credentials.cjs');

let window;
let service;
let credentials;

for (const flag of ['disable-background-networking', 'disable-component-update', 'disable-sync']) {
  app.commandLine.appendSwitch(flag);
}

function answer(operation) {
  try {
    return { ok: true, data: operation() };
  } catch (error) {
    const known = error instanceof ModelError || error instanceof StoreError;
    const code = known ? error.code : 'UNEXPECTED_ERROR';
    console.error(`Atlas Mail operation failed: ${code}`);
    return { ok: false, code };
  }
}

function authorized(event) {
  return Boolean(window && event.sender === window.webContents && event.senderFrame === window.webContents.mainFrame);
}

function createWindow() {
  window = new BrowserWindow({
    width: 1180,
    height: 800,
    minWidth: 720,
    minHeight: 600,
    backgroundColor: '#f5f7fb',
    title: 'Atlas Mail',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true,
    },
  });
  window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  window.webContents.on('will-navigate', (event) => event.preventDefault());
  window.loadFile(path.join(__dirname, '..', 'ui', 'index.html')).catch(() => {
    console.error('Atlas Mail could not load its interface.');
    dialog.showErrorBox('Atlas Mail', 'Die Oberfläche konnte nicht geladen werden. / The application interface could not be loaded.');
    app.quit();
  });
  window.on('closed', () => { window = null; });
}

if (app.commandLine.hasSwitch('no-sandbox')) {
  app.whenReady().then(() => {
    console.error('Atlas Mail refused to start because its sandbox is disabled.');
    dialog.showErrorBox(
      'Atlas Mail',
      'Der Start ohne Sandbox ist nicht zulässig. / Starting without the sandbox is not allowed.',
    );
    app.exit(1);
  });
} else if (!app.requestSingleInstanceLock()) {
  app.quit();
} else {
  app.whenReady().then(() => {
    session.defaultSession.webRequest.onBeforeRequest((details, callback) => {
      const localFile = details.url.startsWith('file:///') && !details.url.startsWith('file:////');
      callback({ cancel: !localFile });
    });
    const directory = dataDirectory();
    service = new MailService(new StateStore(directory));
    credentials = new CredentialStore(path.join(directory, 'credentials'), safeStorage);
    ipcMain.handle('atlas:status', (event) => {
      if (!authorized(event)) return { ok: false, code: 'ACCESS_DENIED' };
      return answer(() => service.status());
    });
    ipcMain.handle('atlas:add-account', (event, input) => {
      if (!authorized(event)) return { ok: false, code: 'ACCESS_DENIED' };
      return answer(() => service.addAccount(input));
    });
    createWindow();
    app.on('activate', () => { if (!window) createWindow(); });
  }).catch((error) => {
    console.error(`Atlas Mail could not start: ${error.code || 'CONFIGURATION_ERROR'}`);
    dialog.showErrorBox('Atlas Mail', 'Start fehlgeschlagen. Datenverzeichnis prüfen. / Could not start. Check the data directory configuration.');
    app.quit();
  });
  app.on('second-instance', () => {
    if (window) {
      if (window.isMinimized()) window.restore();
      window.focus();
    }
  });
  app.on('window-all-closed', () => { if (process.platform !== 'darwin') app.quit(); });
}
