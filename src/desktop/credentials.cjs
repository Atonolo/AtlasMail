'use strict';

const fs = require('node:fs');
const path = require('node:path');
const { assertIdentifier } = require('../core/model.cjs');
const { atomicWrite, withLock } = require('../core/store.cjs');

class CredentialError extends Error {
  constructor(code, cause) {
    super(code, { cause });
    this.name = 'CredentialError';
    this.code = code;
  }
}

class CredentialStore {
  constructor(directory, safeStorage, platform = process.platform) {
    this.directory = directory;
    this.safeStorage = safeStorage;
    this.platform = platform;
  }

  assertAvailable() {
    if (!this.safeStorage.isEncryptionAvailable()) {
      throw new CredentialError('CREDENTIAL_STORE_UNAVAILABLE');
    }
    if (this.platform === 'linux') {
      const backend = this.safeStorage.getSelectedStorageBackend();
      if (!['gnome_libsecret', 'kwallet', 'kwallet5', 'kwallet6'].includes(backend)) {
        throw new CredentialError('CREDENTIAL_STORE_UNAVAILABLE');
      }
    }
  }

  filename(accountId) {
    assertIdentifier(accountId);
    return path.join(this.directory, `${accountId}.bin`);
  }

  put(accountId, secret) {
    const filename = this.filename(accountId);
    if (typeof secret !== 'string' || !secret.length || secret.length > 8192) {
      throw new CredentialError('INVALID_CREDENTIAL');
    }
    this.assertAvailable();
    let encrypted;
    try {
      encrypted = this.safeStorage.encryptString(secret);
    } catch (error) {
      throw new CredentialError('CREDENTIAL_ENCRYPTION_FAILED', error);
    }
    withLock(filename, () => atomicWrite(filename, encrypted));
  }

  get(accountId) {
    const filename = this.filename(accountId);
    this.assertAvailable();
    let encrypted;
    try {
      encrypted = fs.readFileSync(filename);
    } catch (error) {
      if (error.code === 'ENOENT') return null;
      throw new CredentialError('CREDENTIAL_READ_FAILED', error);
    }
    try {
      return this.safeStorage.decryptString(encrypted);
    } catch (error) {
      throw new CredentialError('CREDENTIAL_DECRYPTION_FAILED', error);
    }
  }

  remove(accountId) {
    const filename = this.filename(accountId);
    return withLock(filename, () => {
      try {
        fs.unlinkSync(filename);
        return true;
      } catch (error) {
        if (error.code === 'ENOENT') return false;
        throw new CredentialError('CREDENTIAL_DELETE_FAILED', error);
      }
    });
  }
}

module.exports = { CredentialError, CredentialStore };
