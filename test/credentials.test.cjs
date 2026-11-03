'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { CredentialStore } = require('../src/desktop/credentials.cjs');

const id = '00000000-0000-4000-8000-000000000001';
const fakeStorage = {
  isEncryptionAvailable: () => true,
  getSelectedStorageBackend: () => 'gnome_libsecret',
  encryptString: (value) => Buffer.from([...Buffer.from(value)].map((byte) => byte ^ 0x55)),
  decryptString: (value) => Buffer.from([...value].map((byte) => byte ^ 0x55)).toString(),
};

function temporaryDirectory(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-credentials-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  return directory;
}

test('credential storage uses the injected OS encryption provider and can delete', (t) => {
  const directory = temporaryDirectory(t);
  const store = new CredentialStore(directory, fakeStorage, 'linux');
  const secret = 'synthetic secret';
  store.put(id, secret);
  assert.notEqual(fs.readFileSync(path.join(directory, `${id}.bin`), 'utf8'), secret);
  assert.equal(store.get(id), secret);
  assert.equal(store.remove(id), true);
  assert.equal(store.get(id), null);
});

test('Linux plaintext fallback and an unavailable key store fail closed', (t) => {
  const directory = temporaryDirectory(t);
  for (const storage of [
    { ...fakeStorage, getSelectedStorageBackend: () => 'basic_text' },
    { ...fakeStorage, isEncryptionAvailable: () => false },
  ]) {
    const store = new CredentialStore(directory, storage, 'linux');
    assert.throws(() => store.put(id, 'synthetic'), { code: 'CREDENTIAL_STORE_UNAVAILABLE' });
    assert.equal(fs.existsSync(path.join(directory, `${id}.bin`)), false);
  }
});
