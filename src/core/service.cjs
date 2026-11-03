'use strict';

const { createAdapters } = require('./adapters.cjs');

class MailService {
  constructor(store) {
    this.store = store;
  }

  status() {
    const state = this.store.read();
    return {
      contractVersion: 1,
      schemaVersion: state.schemaVersion,
      accounts: state.accounts,
      adapters: Object.values(createAdapters()).map((adapter) => adapter.capabilities()),
      credentialStore: 'desktop-only',
    };
  }

  addAccount(input) {
    return this.store.addAccount(input);
  }
}

module.exports = { MailService };
