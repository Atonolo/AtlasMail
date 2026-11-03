'use strict';

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('atlasDesktop', Object.freeze({
  status: () => ipcRenderer.invoke('atlas:status'),
  addAccount: (input) => ipcRenderer.invoke('atlas:add-account', input),
}));
