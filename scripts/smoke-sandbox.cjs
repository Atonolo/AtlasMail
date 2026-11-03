'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-sandbox-smoke-'));
try {
  const result = spawnSync(require('electron'), [path.join(__dirname, 'smoke-sandbox-child.cjs'), '--no-sandbox'], {
    encoding: 'utf8',
    env: { ...process.env, ATLAS_MAIL_SMOKE_DIR: directory },
    timeout: 30_000,
  });
  if (result.error) throw result.error;
  assert.equal(result.status, 1, result.stderr);
  assert.match(result.stderr, /refused to start because its sandbox is disabled/);
  assert.match(result.stdout, /Sandbox rejection was explained after application readiness/);
  assert.equal(fs.existsSync(path.join(directory, 'data')), false);
  console.log('Disabled sandbox was rejected before opening the application or writing data.');
} finally {
  fs.rmSync(directory, { recursive: true, force: true });
}
