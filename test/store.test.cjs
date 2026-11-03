'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { StateStore } = require('../src/core/store.cjs');

function temporaryStore(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-store-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  return { directory, store: new StateStore(directory) };
}

const account = (label) => ({ label, address: `${label.toLowerCase()}@example.invalid`, protocol: 'imap-smtp' });

test('synthetic configuration survives restart and validated backup restore', (t) => {
  const { directory, store } = temporaryStore(t);
  const created = store.addAccount(account('First'));
  const backup = store.exportBackup();
  store.addAccount(account('Second'));
  const restarted = new StateStore(directory);
  assert.equal(restarted.read().accounts.length, 2);
  restarted.restoreBackup(backup);
  assert.deepEqual(new StateStore(directory).read().accounts, [created]);
});

test('failed write preserves the last valid bytes and releases the lock', (t) => {
  const { store } = temporaryStore(t);
  store.addAccount(account('First'));
  const before = fs.readFileSync(store.filename);
  const originalRename = fs.renameSync;
  fs.renameSync = () => {
    const error = new Error('injected');
    error.code = 'EIO';
    throw error;
  };
  try {
    assert.throws(() => store.addAccount(account('Second')), { code: 'WRITE_FAILED' });
  } finally {
    fs.renameSync = originalRename;
  }
  assert.deepEqual(fs.readFileSync(store.filename), before);
  assert.equal(fs.existsSync(`${store.filename}.lock`), false);
  assert.equal(store.read().accounts.length, 1);
});

test('temporary-file cleanup failures do not mask the write error or change valid state', (t) => {
  const { store } = temporaryStore(t);
  store.addAccount(account('First'));
  const before = fs.readFileSync(store.filename);
  const originalRename = fs.renameSync;
  const originalUnlink = fs.unlinkSync;
  fs.renameSync = () => { throw new Error('injected rename failure'); };
  fs.unlinkSync = (filename) => {
    if (String(filename).endsWith('.tmp')) throw new Error('injected cleanup failure');
    return originalUnlink(filename);
  };
  try {
    assert.throws(() => store.addAccount(account('Second')), (error) => (
      error.code === 'WRITE_FAILED' && error.cause.errors.length === 2
    ));
  } finally {
    fs.renameSync = originalRename;
    fs.unlinkSync = originalUnlink;
  }
  assert.deepEqual(fs.readFileSync(store.filename), before);
  assert.equal(fs.existsSync(`${store.filename}.lock`), false);
});

test('unsupported migration, invalid restore and corrupt data never overwrite a file', (t) => {
  const { store } = temporaryStore(t);
  store.addAccount(account('First'));
  const before = fs.readFileSync(store.filename);
  assert.throws(() => store.restoreBackup('{"schemaVersion":2}'), { code: 'UNSUPPORTED_SCHEMA' });
  assert.deepEqual(fs.readFileSync(store.filename), before);

  const newer = JSON.parse(before);
  newer.schemaVersion = 2;
  fs.writeFileSync(store.filename, JSON.stringify(newer));
  const newerBytes = fs.readFileSync(store.filename);
  assert.throws(() => store.addAccount(account('Second')), { code: 'UNSUPPORTED_SCHEMA' });
  assert.deepEqual(fs.readFileSync(store.filename), newerBytes);

  fs.writeFileSync(store.filename, '{broken');
  assert.throws(() => store.read(), { code: 'INVALID_STORED_DATA' });
  assert.throws(() => store.addAccount(account('Third')), { code: 'INVALID_STORED_DATA' });
  assert.equal(fs.readFileSync(store.filename, 'utf8'), '{broken');
});

test('an existing lock fails explicitly instead of overwriting concurrent data', (t) => {
  const { store } = temporaryStore(t);
  store.addAccount(account('First'));
  const before = fs.readFileSync(store.filename);
  fs.writeFileSync(`${store.filename}.lock`, 'other process');
  assert.throws(() => store.addAccount(account('Second')), { code: 'STORE_LOCKED' });
  assert.deepEqual(fs.readFileSync(store.filename), before);
});

test('a failed lock cleanup reports that the commit may have succeeded', (t) => {
  const { store } = temporaryStore(t);
  store.addAccount(account('First'));
  const originalUnlink = fs.unlinkSync;
  fs.unlinkSync = (filename) => {
    if (filename === `${store.filename}.lock`) {
      const error = new Error('injected');
      error.code = 'EACCES';
      throw error;
    }
    return originalUnlink(filename);
  };
  try {
    assert.throws(() => store.addAccount(account('Second')), { code: 'COMMIT_UNCERTAIN' });
  } finally {
    fs.unlinkSync = originalUnlink;
  }
  assert.equal(store.read().accounts.length, 2);
  assert.equal(fs.existsSync(`${store.filename}.lock`), true);
});
