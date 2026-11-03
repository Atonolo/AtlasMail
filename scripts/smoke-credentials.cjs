'use strict';

const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const { app, safeStorage } = require('electron');
const { CredentialStore } = require('../src/desktop/credentials.cjs');

const directory = path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'credentials');
const expectBasicText = process.env.ATLAS_MAIL_EXPECT_BASIC_TEXT === '1';
app.setPath('userData', path.join(process.env.ATLAS_MAIL_SMOKE_DIR, 'profile'));
if (expectBasicText) app.commandLine.appendSwitch('password-store', 'basic');

app.whenReady().then(() => {
  const store = new CredentialStore(directory, safeStorage);
  const id = crypto.randomUUID();
  const secret = `synthetic-${crypto.randomBytes(24).toString('hex')}`;
  if (expectBasicText) {
    assert.equal(process.platform, 'linux', 'The basic-text check requires Linux');
    assert.equal(safeStorage.getSelectedStorageBackend(), 'basic_text');
    assert.throws(() => store.put(id, secret), { code: 'CREDENTIAL_STORE_UNAVAILABLE' });
    assert.equal(fs.existsSync(directory), false);
    console.log('Linux basic-text credential fallback was rejected without writing a secret.');
    app.exit(0);
    return;
  }
  store.put(id, secret);
  const backend = process.platform === 'linux' ? safeStorage.getSelectedStorageBackend() : process.platform;
  const ciphertext = fs.readFileSync(path.join(directory, `${id}.bin`));
  assert.equal(ciphertext.includes(Buffer.from(secret)), false);
  assert.equal(store.get(id), secret);
  assert.equal(store.remove(id), true);
  assert.equal(store.get(id), null);
  console.log(`Native credential encryption, retrieval and deletion smoke passed (${backend}).`);
  app.exit(0);
}).catch((error) => {
  console.error(`Native credential smoke failed: ${error.code || error.message}`);
  app.exit(1);
});
