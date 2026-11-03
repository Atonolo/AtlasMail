'use strict';

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawn } = require('node:child_process');

const mode = process.argv[2];
if (!['desktop', 'web', 'credentials'].includes(mode)) {
  console.error('Choose the desktop, web or credentials smoke test.');
  process.exitCode = 1;
} else {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), `atlas-mail-${mode}-smoke-`));
  let child;
  try {
    child = spawn(require('electron'), [path.join(__dirname, `smoke-${mode}.cjs`)], {
      stdio: 'inherit',
      env: { ...process.env, ATLAS_MAIL_SMOKE_DIR: directory },
    });
  } catch (error) {
    fs.rmSync(directory, { recursive: true, force: true });
    throw error;
  }
  let finished = false;
  function finish(code) {
    if (finished) return;
    finished = true;
    fs.rmSync(directory, { recursive: true, force: true });
    process.exitCode = code;
  }
  child.on('error', (error) => {
    console.error(`Smoke test could not start: ${error.message}`);
    finish(1);
  });
  child.on('exit', (code) => finish(code === 0 ? 0 : 1));
}
