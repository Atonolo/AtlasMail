'use strict';

const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { distributionDigest, fileDigest } = require('./digests.cjs');
const { firefoxReportMatches, firefoxReportPath, firefoxRuntimeIdentity } = require('./firefox-runtime.cjs');

const ALLOWED = new Set([
  '0BSD', 'Apache-2.0', 'BlueOak-1.0.0', 'BSD-2-Clause', 'BSD-3-Clause',
  'ISC', 'MIT', 'MPL-2.0', 'Python-2.0', 'Unlicense', 'Zlib',
]);
const NON_OSI_ALTERNATIVES = new Set(['CC0-1.0', 'WTFPL']);

function allowedExpression(expression) {
  if (typeof expression !== 'string' || expression.length > 512 || !/^[A-Za-z0-9.+()\s-]+$/.test(expression)) return false;
  const tokens = expression.match(/[A-Za-z0-9][A-Za-z0-9.+-]*|[()]/g) || [];
  if (tokens.join('') !== expression.replace(/\s+/g, '')) return false;
  let position = 0;
  function term() {
    if (tokens[position] === '(') {
      position += 1;
      const value = disjunction();
      if (value === null || tokens[position] !== ')') return null;
      position += 1;
      return value;
    }
    const token = tokens[position++];
    if (ALLOWED.has(token)) return true;
    if (NON_OSI_ALTERNATIVES.has(token)) return false;
    return null;
  }
  function conjunction() {
    let value = term();
    if (value === null) return null;
    while (tokens[position] === 'AND') {
      position += 1;
      const next = term();
      if (next === null) return null;
      value = value && next;
    }
    return value;
  }
  function disjunction() {
    let value = conjunction();
    if (value === null) return null;
    while (tokens[position] === 'OR') {
      position += 1;
      const next = conjunction();
      if (next === null) return null;
      value = value || next;
    }
    return value;
  }
  return disjunction() === true && position === tokens.length;
}

function scan(directory) {
  const entries = [];
  const visited = new Set();
  function visitModules(modules) {
    if (!fs.existsSync(modules)) return;
    for (const entry of fs.readdirSync(modules, { withFileTypes: true })) {
      if (entry.name.startsWith('.')) continue;
      const location = path.join(modules, entry.name);
      if (entry.name.startsWith('@')) {
        for (const scoped of fs.readdirSync(location, { withFileTypes: true })) {
          visitPackage(path.join(location, scoped.name));
        }
      } else {
        visitPackage(location);
      }
    }
  }
  function visitPackage(location) {
    const resolved = fs.realpathSync(location);
    if (visited.has(resolved)) return;
    visited.add(resolved);
    const manifest = JSON.parse(fs.readFileSync(path.join(location, 'package.json'), 'utf8'));
    entries.push({ name: manifest.name, version: manifest.version, license: manifest.license || null, path: location });
    visitModules(path.join(location, 'node_modules'));
  }
  visitModules(directory);
  return entries;
}

function findKnownBinaryIssues(notices) {
  const products = notices.split('<div class="product">').slice(1);
  if (!products.length) throw new Error('Unrecognized Electron bundled-license notice format');
  const issues = [];
  const fft = products.some((product) => (
    product.includes('2-dim General Purpose FFT (Fast Fourier/Cosine/Sine Transform) Package') &&
    product.includes('You may distribute this ORIGINAL package.')
  ));
  if (fft) {
    issues.push({
      name: 'Electron/Chromium: Ooura FFT',
      license: 'LicenseRef-scancode-ooura-2001',
      reason: 'The bundled license is not OSI-approved.',
      evidence: 'https://scancode-licensedb.aboutcode.org/ooura-2001.html',
    });
  }
  const jpeg = products.some((product) => (
    product.includes('<span class="title">libjpeg-turbo</span>') &&
    product.includes('The IJG (Independent JPEG Group) License') &&
    product.includes('This license applies to the libjpeg API library')
  ));
  if (jpeg) {
    issues.push({
      name: 'Electron/Chromium: libjpeg-turbo libjpeg API',
      license: 'IJG',
      reason: 'The IJG license is not OSI-approved; the BSD license covers different code.',
      evidence: 'https://raw.githubusercontent.com/spdx/license-list-data/main/json/details/IJG.json',
    });
  }
  const sqlite = products.some((product) => (
    product.includes('<span class="title">sqlite</span>') &&
    product.includes('The author disclaims copyright to this source code.')
  ));
  if (sqlite) {
    issues.push({
      name: 'Electron/Chromium: SQLite',
      license: 'blessing',
      reason: 'SQLite is public domain, not distributed under an OSI-approved license.',
      evidence: 'https://sqlite.org/copyright.html',
    });
  }
  return issues;
}

function findKnownNodeIssues(versions) {
  if (!versions.sqlite) return [];
  return [{
    name: 'Node.js runtime: SQLite',
    license: 'blessing',
    reason: 'The Node runtime includes SQLite, which is not distributed under an OSI-approved license.',
    evidence: 'https://sqlite.org/copyright.html',
    sqliteVersion: versions.sqlite,
  }];
}

function findKnownFirefoxIssues(identity) {
  const jpeg = new Map([
    ['0e90c1ba93fb49a0761da9463f6a175976fb7f4a1767e5f8e41b1798ea1f1c48', '153.6.0 ESR'],
    ['921dcc79f7867c3b579fd2f655655a0bcac04b81d47ab3730918c729cc479b95', '159.0.1'],
    ['4c2cbbfb8a0b85effa9a623fd3ffd293db3fb16bcacf5f1bf9acc7aa32ef02fc', '153.6.0 ESR'],
  ]).get(identity.firefoxCoreSha256);
  const windowsSqlite = new Map([
    ['79beaf1fe77f08f5e7a0f391c3235c68d6d1d9f416be66f40afd370b95741e74', '153.6.0 ESR'],
    ['afd1e2388a9383298e1691f23aef1ddf93e67ade74aa26b6001695bd0e9c0687', '159.0.1'],
  ]).get(identity.firefoxNssSha256);
  const linuxSqlite = new Map([
    ['ccb6bf36aa35bea2b0cc0fff4177c0c25987f2f63438d77d1369afe97b65ece3', '153.6.0 ESR'],
  ]).get(identity.firefoxSqliteSha256);
  const issues = [];
  if (jpeg) issues.push({
    name: 'Firefox test runtime: libjpeg-turbo libjpeg API',
    version: jpeg,
    license: 'IJG',
    reason: 'Matched Firefox symbols map the runtime to IJG-licensed libjpeg API code.',
    evidence: 'THIRD_PARTY_NOTICES.md',
  });
  if (windowsSqlite || linuxSqlite) issues.push({
    name: 'Firefox test runtime: SQLite',
    version: windowsSqlite || linuxSqlite,
    license: 'blessing',
    reason: 'Matched Firefox library symbols map the runtime to public-domain SQLite code.',
    evidence: 'THIRD_PARTY_NOTICES.md',
  });
  return issues;
}

function binaryAuditAccepted(audit, identity) {
  if (!audit || audit.schemaVersion !== 3 || !Array.isArray(audit.targets) || !identity) return false;
  const keys = [
    'platform', 'arch', 'electronVersion', 'nodeVersion', 'nodeExecutableSha256', 'chromiumNoticesSha256',
    'electronDistributionSha256', 'packageLockSha256', 'firefoxExecutableSha256',
    'firefoxCoreSha256', 'firefoxDistributionSha256',
  ];
  if (!keys.every((key) => typeof identity[key] === 'string' && identity[key].length > 0)) return false;
  const matching = audit.targets.filter((target) => (
    target && typeof target === 'object' && keys.every((key) => target[key] === identity[key])
  ));
  return matching.length === 1 &&
    typeof matching[0].evidence === 'string' && matching[0].evidence.trim().length > 0;
}

function main() {
  const directory = path.resolve('node_modules');
  if (!fs.existsSync(directory)) throw new Error('Install dependencies before checking licenses');
  const entries = scan(directory);
  if (!entries.length) throw new Error('No installed dependencies found');
  const rejected = entries.filter((entry) => (
    typeof entry.name !== 'string' || typeof entry.version !== 'string' || !allowedExpression(entry.license)
  ));
  const electronDirectory = path.join(directory, 'electron');
  const electron = JSON.parse(fs.readFileSync(path.join(electronDirectory, 'package.json'), 'utf8'));
  const bundledNotices = path.join(electronDirectory, 'dist', 'LICENSES.chromium.html');
  const auditPath = path.resolve('licenses', 'binary-audit.json');
  const lockfile = fs.readFileSync(path.resolve('package-lock.json'));
  const noticeBytes = fs.readFileSync(bundledNotices);
  const hash = crypto.createHash('sha256').update(noticeBytes).digest('hex');
  const identity = {
    platform: process.platform,
    arch: process.arch,
    electronVersion: electron.version,
    nodeVersion: process.version,
    nodeExecutableSha256: fileDigest(process.execPath),
    chromiumNoticesSha256: hash,
    electronDistributionSha256: distributionDigest(path.join(electronDirectory, 'dist')),
    packageLockSha256: crypto.createHash('sha256').update(lockfile).digest('hex'),
  };
  rejected.push(...findKnownBinaryIssues(noticeBytes.toString('utf8')).map((issue) => ({
    ...issue,
    version: electron.version,
  })));
  rejected.push(...findKnownNodeIssues(process.versions).map((issue) => ({
    ...issue,
    version: process.version,
  })));
  let firefox;
  try {
    firefox = firefoxRuntimeIdentity();
    identity.firefoxExecutableSha256 = firefox.firefoxExecutableSha256;
    identity.firefoxCoreSha256 = firefox.firefoxCoreSha256;
    identity.firefoxDistributionSha256 = firefox.firefoxDistributionSha256;
    rejected.push(...findKnownFirefoxIssues(firefox));
  } catch (error) {
    rejected.push({ name: 'Firefox test runtime', license: 'UNREVIEWED', reason: error.message });
  }
  if (firefox) {
    try {
      const report = JSON.parse(fs.readFileSync(firefoxReportPath(), 'utf8'));
      if (!firefoxReportMatches(report, firefox)) throw new Error('Firefox smoke report does not match the current runtime');
    } catch (error) {
      rejected.push({ name: 'Firefox smoke runtime evidence', license: 'UNREVIEWED', reason: error.message });
    }
  }
  const audit = fs.existsSync(auditPath) ? JSON.parse(fs.readFileSync(auditPath, 'utf8')) : null;
  if (!binaryAuditAccepted(audit, identity)) {
    rejected.push({
      name: 'Electron, Firefox and build/test-tool binaries',
      version: electron.version,
      license: 'UNREVIEWED',
      ...identity,
    });
  }
  console.log(JSON.stringify({ checked: entries, rejected }, null, 2));
  if (rejected.length) process.exitCode = 1;
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`License check failed: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = {
  allowedExpression, binaryAuditAccepted, distributionDigest, fileDigest,
  findKnownBinaryIssues, findKnownFirefoxIssues, findKnownNodeIssues,
};
