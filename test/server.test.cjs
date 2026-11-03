'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { StateStore } = require('../src/core/store.cjs');
const { MailService } = require('../src/core/service.cjs');
const { createServer } = require('../src/server/server.cjs');

const token = 'A'.repeat(43);

test('web boundary authenticates, rejects cross-origin requests and persists synthetic accounts', async (t) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-web-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const server = createServer({ service: new MailService(new StateStore(directory)), token });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  t.after(() => new Promise((resolve) => server.close(resolve)));
  const base = `http://127.0.0.1:${server.address().port}`;
  const authorization = { Authorization: `Bearer ${token}` };

  const page = await fetch(base);
  assert.equal(page.status, 200);
  assert.match(page.headers.get('content-security-policy'), /script-src 'self'/);
  const icon = await fetch(`${base}/favicon.svg`);
  assert.equal(icon.status, 200);
  assert.match(icon.headers.get('content-type'), /^image\/svg\+xml/);
  assert.equal((await fetch(`${base}/api/v1/status`)).status, 401);
  assert.equal((await fetch(`${base}/api/v1/status`, {
    headers: { ...authorization, Origin: 'https://other.invalid' },
  })).status, 403);
  const create = await fetch(`${base}/api/v1/accounts`, {
    method: 'POST',
    headers: { ...authorization, 'Content-Type': 'application/json' },
    body: JSON.stringify({ label: 'Synthetic', address: 'test@example.invalid', protocol: 'jmap' }),
  });
  assert.equal(create.status, 201);
  const saved = await create.json();
  const status = await (await fetch(`${base}/api/v1/status`, { headers: authorization })).json();
  assert.deepEqual(status.data.accounts, [saved.data]);
  assert.deepEqual(new StateStore(directory).read().accounts, [saved.data]);
  assert.ok(status.data.adapters.every((adapter) => adapter.available === false));
  assert.equal(status.data.credentialStore, 'desktop-only');

  const extra = await fetch(`${base}/api/v1/accounts`, {
    method: 'POST',
    headers: { ...authorization, 'Content-Type': 'application/json' },
    body: JSON.stringify({ label: 'A', address: 'a@example.invalid', protocol: 'jmap', password: 'secret' }),
  });
  assert.equal(extra.status, 400);
  assert.equal(new StateStore(directory).read().accounts.length, 1);
  const oversized = await fetch(`${base}/api/v1/accounts`, {
    method: 'POST',
    headers: { ...authorization, 'Content-Type': 'application/json' },
    body: 'x'.repeat(20 * 1024),
  });
  assert.equal(oversized.status, 413);
});

test('web service refuses weakly shaped tokens and non-HTTPS public origins', () => {
  assert.throws(() => createServer({ service: {}, token: 'short' }), /ATLAS_MAIL_WEB_TOKEN/);
  assert.throws(() => createServer({ service: {}, token, publicOrigin: 'http://example.invalid' }), /HTTPS origin/);
  assert.doesNotThrow(() => createServer({ service: {}, token, publicOrigin: 'https://example.invalid' }));
});
