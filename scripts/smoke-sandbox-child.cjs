'use strict';

const path = require('node:path');
const { app, dialog } = require('electron');

app.setPath('userData', path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'profile'));
process.env.ATLAS_MAIL_DATA_DIR = path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'data');
dialog.showErrorBox = (title, message) => {
  if (!app.isReady()
    || title !== 'Atlas Mail'
    || !message.includes('Starting without the sandbox is not allowed.')) {
    throw new Error('The sandbox rejection was not explained to the user');
  }
  console.log('Sandbox rejection was explained after application readiness.');
};
require('../src/desktop/main.cjs');
