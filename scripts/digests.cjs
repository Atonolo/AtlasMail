'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');

function updateFileHash(hash, filename, expectedSize, buffer) {
  const descriptor = fs.openSync(filename, 'r');
  let total = 0;
  try {
    let bytes;
    while ((bytes = fs.readSync(descriptor, buffer, 0, buffer.length, null)) > 0) {
      hash.update(buffer.subarray(0, bytes));
      total += bytes;
    }
  } finally {
    fs.closeSync(descriptor);
  }
  if (total !== expectedSize) throw new Error(`File changed while hashing: ${filename}`);
}

function fileDigest(filename) {
  const hash = crypto.createHash('sha256');
  updateFileHash(hash, filename, fs.statSync(filename).size, Buffer.allocUnsafe(256 * 1024));
  return hash.digest('hex');
}

function distributionDigest(directory) {
  const root = path.resolve(directory);
  const hash = crypto.createHash('sha256');
  const buffer = Buffer.allocUnsafe(256 * 1024);
  hash.update('atlas-mail-distribution-v1\0');

  function visit(parent, relativeParent) {
    const entries = fs.readdirSync(parent, { withFileTypes: true })
      .sort((left, right) => left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
    for (const entry of entries) {
      const location = path.join(parent, entry.name);
      const relative = relativeParent ? `${relativeParent}/${entry.name}` : entry.name;
      const stat = fs.lstatSync(location);
      if (stat.isDirectory()) {
        hash.update(`directory\0${relative}\0`);
        visit(location, relative);
      } else if (stat.isFile()) {
        hash.update(`file\0${relative}\0${stat.size}\0`);
        updateFileHash(hash, location, stat.size, buffer);
      } else if (stat.isSymbolicLink()) {
        const target = fs.readlinkSync(location);
        const resolved = path.resolve(parent, target);
        const relativeTarget = path.relative(root, resolved);
        if (path.isAbsolute(target) || path.isAbsolute(relativeTarget) ||
            relativeTarget === '..' || relativeTarget.startsWith(`..${path.sep}`)) {
          throw new Error(`Distribution has an external symlink: ${relative}`);
        }
        hash.update(`symlink\0${relative}\0${target}\0`);
      } else {
        throw new Error(`Distribution has an unsupported entry: ${relative}`);
      }
    }
  }

  visit(root, '');
  return hash.digest('hex');
}

module.exports = { distributionDigest, fileDigest };
