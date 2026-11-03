'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { createRequire } = require('node:module');

test('build filename sanitizer handles reserved names and UTF-8 byte limits', () => {
  const fromBuilder = createRequire(require.resolve('electron-builder/package.json'));
  const fromBuilderUtil = createRequire(fromBuilder.resolve('builder-util/package.json'));
  const sanitize = fromBuilderUtil('sanitize-filename');
  assert.equal(sanitize('CON.txt'), '');
  assert.equal(sanitize('bad<name>.txt'), 'badname.txt');
  assert.equal(sanitize(`${'a'.repeat(253)}🎉suffix`), 'a'.repeat(253));
  assert.ok(Buffer.byteLength(sanitize('🎉'.repeat(100))) <= 255);
});
