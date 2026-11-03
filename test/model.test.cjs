'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { emptyState, makeAccount, validateState } = require('../src/core/model.cjs');
const { createAdapters } = require('../src/core/adapters.cjs');

const fixture = () => JSON.parse(fs.readFileSync(path.join(__dirname, 'fixtures', 'state-v1.json'), 'utf8'));

test('versioned reference data validates with stable, linked identifiers', () => {
  const state = fixture();
  assert.equal(validateState(state), state);
  assert.equal(state.schemaVersion, 1);
  assert.equal(state.queue[0].entityId, state.tasks[0].id);
  assert.deepEqual(validateState(emptyState()), emptyState());
});

test('account input rejects secret fields and invalid identities', () => {
  const id = '00000000-0000-4000-8000-000000000009';
  const account = makeAccount({ label: ' Example ', address: 'test@example.invalid ', protocol: 'jmap' }, id);
  assert.equal(account.label, 'Example');
  assert.equal(account.address, 'test@example.invalid');
  assert.throws(() => makeAccount({ label: 'A', address: 'a@example.invalid', protocol: 'jmap', password: 'secret' }, id), { code: 'INVALID_DATA' });
  assert.throws(() => makeAccount({ label: 'A', address: 'invalid', protocol: 'jmap' }, id), { code: 'INVALID_ACCOUNT' });
  assert.throws(() => makeAccount(
    { label: 'A', address: 'a@example.invalid', protocol: 'jmap' },
    'AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA',
  ), { code: 'INVALID_IDENTIFIER' });
});

test('unknown versions, fields, duplicate identifiers and cross-account references fail closed', () => {
  const version = fixture();
  version.schemaVersion = 2;
  assert.throws(() => validateState(version), { code: 'UNSUPPORTED_SCHEMA' });
  const extra = fixture();
  extra.accounts[0].password = 'never-store';
  assert.throws(() => validateState(extra), { code: 'INVALID_DATA' });
  const duplicate = fixture();
  duplicate.tasks[0].id = duplicate.messages[0].id;
  assert.throws(() => validateState(duplicate), { code: 'DUPLICATE_IDENTIFIER' });
  const wrongKind = fixture();
  wrongKind.queue[0].kind = 'event';
  assert.throws(() => validateState(wrongKind), { code: 'INVALID_REFERENCE' });
  const incompleteCache = fixture();
  incompleteCache.cache[0].state = 'complete';
  assert.throws(() => validateState(incompleteCache), { code: 'INVALID_DATA' });
  const foreign = fixture();
  foreign.accounts.push({ id: '00000000-0000-4000-8000-000000000009', label: 'Other', address: 'other@example.invalid', protocol: 'jmap' });
  foreign.queue[0].accountId = foreign.accounts[1].id;
  assert.throws(() => validateState(foreign), { code: 'INVALID_REFERENCE' });
});

test('all future mail and DAV adapters explicitly reject operations', async () => {
  for (const [name, adapter] of Object.entries(createAdapters())) {
    assert.deepEqual(adapter.capabilities(), { contractVersion: 1, protocol: name, available: false });
    for (const operation of ['connect', 'synchronize', 'send']) {
      await assert.rejects(adapter[operation](), { code: 'FEATURE_UNAVAILABLE', feature: name });
    }
  }
});
