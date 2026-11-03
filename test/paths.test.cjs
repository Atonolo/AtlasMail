'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { dataDirectory } = require('../src/core/paths.cjs');

test('relative configuration paths fail instead of silently selecting another directory', () => {
  assert.throws(() => dataDirectory({ ATLAS_MAIL_DATA_DIR: 'relative' }), /absolute path/);
  assert.throws(() => dataDirectory({ XDG_DATA_HOME: 'relative' }, 'linux'), /absolute path/);
  assert.throws(() => dataDirectory({ APPDATA: 'relative' }, 'win32'), /absolute path/);
});
