'use strict';

const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { ModelError, emptyState, makeAccount, validateState } = require('./model.cjs');

const MAX_FILE_BYTES = 10 * 1024 * 1024;

class StoreError extends Error {
  constructor(code, cause) {
    super(code, { cause });
    this.name = 'StoreError';
    this.code = code;
  }
}

function ensureDirectory(directory) {
  try {
    fs.mkdirSync(directory, { recursive: true, mode: 0o700 });
  } catch (error) {
    throw new StoreError('STORAGE_UNAVAILABLE', error);
  }
}

function withLock(filename, operation) {
  ensureDirectory(path.dirname(filename));
  const lock = `${filename}.lock`;
  let descriptor;
  try {
    descriptor = fs.openSync(lock, 'wx', 0o600);
  } catch (error) {
    throw new StoreError(error.code === 'EEXIST' ? 'STORE_LOCKED' : 'STORAGE_UNAVAILABLE', error);
  }
  let result;
  let operationError;
  let operationStarted = false;
  try {
    try {
      fs.writeFileSync(descriptor, String(process.pid), 'utf8');
      fs.fsyncSync(descriptor);
    } catch (error) {
      throw new StoreError('STORAGE_UNAVAILABLE', error);
    }
    operationStarted = true;
    result = operation();
  } catch (error) {
    operationError = error;
  }
  const cleanupErrors = [];
  try {
    fs.closeSync(descriptor);
  } catch (error) {
    cleanupErrors.push(error);
  }
  try {
    fs.unlinkSync(lock);
  } catch (error) {
    cleanupErrors.push(error);
  }
  if (cleanupErrors.length) {
    if (operationError) cleanupErrors.unshift(operationError);
    throw new StoreError(
      operationStarted ? 'COMMIT_UNCERTAIN' : 'LOCK_CLEANUP_FAILED',
      new AggregateError(cleanupErrors),
    );
  }
  if (operationError) throw operationError;
  return result;
}

function atomicWrite(filename, content) {
  const temporary = `${filename}.${crypto.randomUUID()}.tmp`;
  let descriptor;
  let exists = false;
  const errors = [];
  try {
    descriptor = fs.openSync(temporary, 'wx', 0o600);
    exists = true;
    fs.writeFileSync(descriptor, content);
    fs.fsyncSync(descriptor);
    fs.closeSync(descriptor);
    descriptor = undefined;
    fs.renameSync(temporary, filename);
    exists = false;
  } catch (error) {
    errors.push(error);
  } finally {
    if (descriptor !== undefined) {
      try {
        fs.closeSync(descriptor);
      } catch (error) {
        errors.push(error);
      }
    }
    if (exists) {
      try {
        fs.unlinkSync(temporary);
      } catch (error) {
        errors.push(error);
      }
    }
  }
  if (errors.length) throw new StoreError('WRITE_FAILED', new AggregateError(errors));
}

function parseState(content) {
  if (Buffer.byteLength(content, 'utf8') > MAX_FILE_BYTES) {
    throw new StoreError('DATA_TOO_LARGE');
  }
  try {
    return validateState(JSON.parse(content));
  } catch (error) {
    if (error instanceof ModelError && error.code === 'UNSUPPORTED_SCHEMA') {
      throw new StoreError('UNSUPPORTED_SCHEMA', error);
    }
    throw new StoreError('INVALID_STORED_DATA', error);
  }
}

class StateStore {
  constructor(directory) {
    this.filename = path.join(directory, 'state-v1.json');
  }

  read() {
    let descriptor;
    try {
      descriptor = fs.openSync(this.filename, 'r');
    } catch (error) {
      if (error.code === 'ENOENT') return emptyState();
      throw new StoreError('READ_FAILED', error);
    }
    let content;
    try {
      if (fs.fstatSync(descriptor).size > MAX_FILE_BYTES) {
        throw new StoreError('DATA_TOO_LARGE');
      }
      content = fs.readFileSync(descriptor, 'utf8');
    } catch (error) {
      if (error instanceof StoreError) throw error;
      throw new StoreError('READ_FAILED', error);
    } finally {
      fs.closeSync(descriptor);
    }
    return parseState(content);
  }

  update(change) {
    return withLock(this.filename, () => {
      const current = this.read();
      const updated = change(current);
      validateState(updated);
      const content = `${JSON.stringify(updated, null, 2)}\n`;
      if (Buffer.byteLength(content, 'utf8') > MAX_FILE_BYTES) {
        throw new StoreError('DATA_TOO_LARGE');
      }
      atomicWrite(this.filename, content);
      return updated;
    });
  }

  addAccount(input) {
    const account = makeAccount(input, crypto.randomUUID());
    this.update((state) => ({ ...state, accounts: [...state.accounts, account] }));
    return account;
  }

  exportBackup() {
    return `${JSON.stringify(this.read(), null, 2)}\n`;
  }

  restoreBackup(content) {
    if (typeof content !== 'string') throw new StoreError('INVALID_STORED_DATA');
    const restored = parseState(content);
    this.update(() => restored);
    return restored;
  }
}

module.exports = { StateStore, StoreError, atomicWrite, parseState, withLock };
