'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const {
  allowedExpression, binaryAuditAccepted, distributionDigest, fileDigest,
  findKnownBinaryIssues, findKnownFirefoxIssues, findKnownNodeIssues,
} = require('../scripts/check-licenses.cjs');
const {
  firefoxArguments, firefoxExecutable, firefoxReportMatches, firefoxRuntimeIdentity,
} = require('../scripts/firefox-runtime.cjs');

const hostIssues = findKnownNodeIssues(process.versions).map((entry) => entry.license);
const hostNodeDigest = fileDigest(process.execPath);

test('license allowlist accepts only reviewed OSI-compatible SPDX choices', () => {
  assert.equal(allowedExpression('MIT'), true);
  assert.equal(allowedExpression('Python-2.0'), true);
  assert.equal(allowedExpression('(MIT OR Apache-2.0) AND BSD-3-Clause'), true);
  assert.equal(allowedExpression('WTFPL OR ISC'), true);
  assert.equal(allowedExpression('(MIT OR CC0-1.0) AND BSD-3-Clause'), true);
  assert.equal(allowedExpression('WTFPL AND MIT OR ISC'), true);
  assert.equal(allowedExpression('WTFPL'), false);
  assert.equal(allowedExpression('CC0-1.0'), false);
  assert.equal(allowedExpression('WTFPL AND ISC'), false);
  assert.equal(allowedExpression('MIT OR Proprietary'), false);
  assert.equal(allowedExpression('MIT OR'), false);
  assert.equal(allowedExpression('MIT -'), false);
  assert.equal(allowedExpression('MIT +'), false);
  assert.equal(allowedExpression('(MIT'), false);
  assert.equal(allowedExpression(`${'('.repeat(300)}MIT${')'.repeat(300)}`), false);
  assert.equal(allowedExpression(''), false);
  assert.equal(allowedExpression(null), false);
});

test('bundled notices block known non-OSI FFT, JPEG and SQLite terms and unknown formats', () => {
  const fft = '<div class="product"><span class="title">2-dim General Purpose FFT (Fast Fourier/Cosine/Sine Transform) Package</span><pre>You may distribute this ORIGINAL package.</pre></div>';
  const jpeg = '<div class="product"><span class="title">libjpeg-turbo</span><pre>The IJG (Independent JPEG Group) License\nThis license applies to the libjpeg API library</pre></div>';
  const sqlite = '<div class="product"><span class="title">sqlite</span><pre>The author disclaims copyright to this source code.</pre></div>';
  assert.equal(findKnownBinaryIssues(fft)[0].license, 'LicenseRef-scancode-ooura-2001');
  assert.equal(findKnownBinaryIssues(jpeg)[0].license, 'IJG');
  assert.equal(findKnownBinaryIssues(sqlite)[0].license, 'blessing');
  assert.deepEqual(findKnownBinaryIssues(fft + jpeg + sqlite).map((entry) => entry.license), [
    'LicenseRef-scancode-ooura-2001', 'IJG', 'blessing',
  ]);
  assert.deepEqual(findKnownBinaryIssues('<div class="product"><span>MIT component</span></div>'), []);
  assert.deepEqual(findKnownBinaryIssues('<div class="product">2-dim General Purpose FFT (Fast Fourier/Cosine/Sine Transform) Package</div><div class="product">You may distribute this ORIGINAL package.</div>'), []);
  assert.deepEqual(findKnownBinaryIssues('<div class="product"><span class="title">libjpeg-turbo</span></div><div class="product">The IJG (Independent JPEG Group) License\nThis license applies to the libjpeg API library</div>'), []);
  assert.deepEqual(findKnownBinaryIssues('<div class="product"><span class="title">sqlite</span></div><div class="product">The author disclaims copyright to this source code.</div>'), []);
  assert.throws(() => findKnownBinaryIssues('unknown notice format'), /Unrecognized/);
});

test('binary reviews must match the target, runtime and exact reviewed inputs', () => {
  const identity = {
    platform: 'linux',
    arch: 'x64',
    electronVersion: '1.0.0',
    nodeVersion: 'v1.0.0',
    nodeExecutableSha256: 'node-binary',
    chromiumNoticesSha256: 'notices',
    electronDistributionSha256: 'distribution',
    packageLockSha256: 'lock',
    firefoxExecutableSha256: 'firefox-executable',
    firefoxCoreSha256: 'firefox-core',
    firefoxDistributionSha256: 'firefox-distribution',
  };
  const target = { ...identity, evidence: 'Completed review reference' };
  const audit = { schemaVersion: 3, targets: [{ ...target, platform: 'darwin' }, target] };
  assert.equal(binaryAuditAccepted(audit, identity), true);
  for (const key of Object.keys(identity)) {
    assert.equal(binaryAuditAccepted(audit, { ...identity, [key]: 'different' }), false, key);
  }
  assert.equal(binaryAuditAccepted(null, identity), false);
  assert.equal(binaryAuditAccepted({ schemaVersion: 2, targets: [target] }, identity), false);
  assert.equal(binaryAuditAccepted({ schemaVersion: 3, targets: [target, target] }, identity), false);
  assert.equal(binaryAuditAccepted({ schemaVersion: 3, targets: [{ ...target, evidence: '  ' }] }, identity), false);
  assert.equal(binaryAuditAccepted(audit, { ...identity, firefoxCoreSha256: undefined }), false);
});

test('Node SQLite is rejected even independently of Chromium notices', () => {
  assert.deepEqual(findKnownNodeIssues({}), []);
  const issues = findKnownNodeIssues({ sqlite: '3.0.0' });
  assert.equal(issues.length, 1);
  assert.equal(issues[0].license, 'blessing');
  assert.equal(issues[0].sqliteVersion, '3.0.0');
});

test('known Firefox test libraries independently block IJG and SQLite', () => {
  const firefox159 = {
    firefoxCoreSha256: '921dcc79f7867c3b579fd2f655655a0bcac04b81d47ab3730918c729cc479b95',
    firefoxNssSha256: 'afd1e2388a9383298e1691f23aef1ddf93e67ade74aa26b6001695bd0e9c0687',
  };
  const firefoxEsr = {
    firefoxCoreSha256: '0e90c1ba93fb49a0761da9463f6a175976fb7f4a1767e5f8e41b1798ea1f1c48',
    firefoxNssSha256: '79beaf1fe77f08f5e7a0f391c3235c68d6d1d9f416be66f40afd370b95741e74',
  };
  for (const identity of [firefox159, firefoxEsr]) {
    assert.deepEqual(findKnownFirefoxIssues(identity).map((issue) => issue.license), ['IJG', 'blessing']);
    assert.deepEqual(findKnownFirefoxIssues({ ...identity, firefoxCoreSha256: 'different' })
      .map((issue) => issue.license), ['blessing']);
    assert.deepEqual(findKnownFirefoxIssues({ ...identity, firefoxNssSha256: null })
      .map((issue) => issue.license), ['IJG']);
  }
  const linuxEsr = {
    firefoxCoreSha256: '4c2cbbfb8a0b85effa9a623fd3ffd293db3fb16bcacf5f1bf9acc7aa32ef02fc',
    firefoxSqliteSha256: 'ccb6bf36aa35bea2b0cc0fff4177c0c25987f2f63438d77d1369afe97b65ece3',
  };
  assert.deepEqual(findKnownFirefoxIssues(linuxEsr).map((issue) => issue.license), ['IJG', 'blessing']);
  assert.deepEqual(findKnownFirefoxIssues({ ...linuxEsr, firefoxCoreSha256: 'different' })
    .map((issue) => issue.license), ['blessing']);
  assert.deepEqual(findKnownFirefoxIssues({ ...linuxEsr, firefoxSqliteSha256: null })
    .map((issue) => issue.license), ['IJG']);
  assert.deepEqual(findKnownFirefoxIssues({}), []);
});

test('Firefox selection requires an absolute override and uses the Windows-only wait flag', () => {
  assert.throws(() => firefoxExecutable({ FIREFOX_BINARY: 'relative-firefox' }), /absolute path/);
  assert.equal(firefoxArguments('profile', 'win32').includes('-wait-for-browser'), true);
  assert.equal(firefoxArguments('profile', 'linux').includes('-wait-for-browser'), false);
  assert.equal(firefoxArguments('profile', 'darwin').includes('-wait-for-browser'), false);
});

test('file digest changes with the reviewed executable', (t) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-binary-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const file = path.join(directory, 'node');
  fs.writeFileSync(file, 'original');
  const original = fileDigest(file);
  fs.writeFileSync(file, 'modified');
  assert.notEqual(fileDigest(file), original);
});

test('distribution digest tracks file names, contents and directory structure', (t) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-distribution-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  fs.mkdirSync(path.join(directory, 'resources'));
  fs.writeFileSync(path.join(directory, 'resources', 'asset'), 'one');
  const original = distributionDigest(directory);
  assert.equal(distributionDigest(directory), original);
  fs.writeFileSync(path.join(directory, 'resources', 'asset'), 'two');
  assert.notEqual(distributionDigest(directory), original);
  fs.writeFileSync(path.join(directory, 'resources', 'asset'), 'one');
  fs.renameSync(path.join(directory, 'resources', 'asset'), path.join(directory, 'resources', 'renamed'));
  assert.notEqual(distributionDigest(directory), original);
});

test('distribution digest rejects symlinks outside the reviewed tree', (t) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-symlink-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const root = path.join(directory, 'dist');
  fs.mkdirSync(root);
  fs.writeFileSync(path.join(root, 'internal'), 'internal');
  const link = path.join(root, 'link');
  try {
    fs.symlinkSync('internal', link, 'file');
  } catch (error) {
    if (process.platform !== 'win32' || error.code !== 'EPERM') throw error;
    t.skip('Creating symbolic links is not permitted on this Windows host');
    return;
  }
  const original = distributionDigest(root);
  fs.unlinkSync(link);
  fs.writeFileSync(path.join(directory, 'external'), 'external');
  fs.symlinkSync(path.join('..', 'external'), link, 'file');
  assert.throws(() => distributionDigest(root), /external symlink/);
  fs.unlinkSync(link);
  fs.symlinkSync('internal', link, 'file');
  assert.equal(distributionDigest(root), original);
});

function reviewedFixture(t, notices) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'atlas-mail-license-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const electron = path.join(directory, 'node_modules', 'electron');
  fs.mkdirSync(path.join(electron, 'dist'), { recursive: true });
  fs.mkdirSync(path.join(directory, 'licenses'));
  fs.mkdirSync(path.join(directory, 'reports'));
  const firefoxDirectory = path.join(directory, 'firefox');
  fs.mkdirSync(firefoxDirectory);
  const firefoxExecutablePath = path.join(firefoxDirectory, process.platform === 'win32' ? 'firefox.exe' : 'firefox');
  const executableHeaders = {
    win32: Buffer.from('4d5a0000', 'hex'),
    linux: Buffer.from('7f454c46', 'hex'),
    darwin: Buffer.from('cffaedfe', 'hex'),
  };
  fs.writeFileSync(firefoxExecutablePath, executableHeaders[process.platform]);
  const coreName = { win32: 'xul.dll', darwin: 'XUL', linux: 'libxul.so' }[process.platform];
  fs.writeFileSync(path.join(firefoxDirectory, coreName), 'synthetic Firefox core');
  const env = { ...process.env, FIREFOX_BINARY: firefoxExecutablePath };
  const firefox = firefoxRuntimeIdentity(env);
  fs.writeFileSync(path.join(directory, 'reports', 'firefox-runtime.json'), JSON.stringify({
    schemaVersion: 2, browserVersion: 'synthetic Firefox', ...firefox,
  }));
  fs.writeFileSync(path.join(electron, 'package.json'), JSON.stringify({ name: 'electron', version: '1.0.0', license: 'MIT' }));
  fs.writeFileSync(path.join(electron, 'dist', 'LICENSES.chromium.html'), notices);
  const lockfile = '{}';
  fs.writeFileSync(path.join(directory, 'package-lock.json'), lockfile);
  fs.writeFileSync(path.join(directory, 'licenses', 'binary-audit.json'), JSON.stringify({
    schemaVersion: 3,
    targets: [{
      platform: process.platform,
      arch: process.arch,
      electronVersion: '1.0.0',
      nodeVersion: process.version,
      nodeExecutableSha256: hostNodeDigest,
      chromiumNoticesSha256: crypto.createHash('sha256').update(notices).digest('hex'),
      electronDistributionSha256: distributionDigest(path.join(electron, 'dist')),
      packageLockSha256: crypto.createHash('sha256').update(lockfile).digest('hex'),
      firefoxExecutableSha256: firefox.firefoxExecutableSha256,
      firefoxCoreSha256: firefox.firefoxCoreSha256,
      firefoxDistributionSha256: firefox.firefoxDistributionSha256,
      evidence: 'Synthetic test fixture only',
    }],
  }));
  const run = () => spawnSync(process.execPath, [path.join(__dirname, '..', 'scripts', 'check-licenses.cjs')], {
    cwd: directory,
    encoding: 'utf8',
    env,
  });
  return { directory, electron, firefox, firefoxDirectory, firefoxExecutablePath, run };
}

test('Firefox smoke evidence matches only the same browser identity and a reported version', (t) => {
  const fixture = reviewedFixture(t, '<div class="product"><span>MIT component</span></div>');
  const report = { schemaVersion: 2, browserVersion: 'test', ...fixture.firefox };
  assert.equal(firefoxReportMatches(report, fixture.firefox), true);
  for (const key of Object.keys(fixture.firefox)) {
    assert.equal(firefoxReportMatches({ ...report, [key]: 'different' }, fixture.firefox), false, key);
  }
  assert.equal(firefoxReportMatches({ ...report, browserVersion: '  ' }, fixture.firefox), false);
  assert.equal(firefoxReportMatches({ ...report, schemaVersion: 1 }, fixture.firefox), false);
  assert.equal(firefoxReportMatches(null, fixture.firefox), false);
  assert.equal(firefoxReportMatches(report, {}), false);
  assert.equal(firefoxReportMatches(report, { ...fixture.firefox, firefoxCoreSha256: undefined }), false);
});

test('Linux Firefox SQLite library changes invalidate smoke evidence', (t) => {
  if (process.platform !== 'linux') {
    t.skip('The separate Firefox SQLite library is specific to the Linux distribution');
    return;
  }
  const fixture = reviewedFixture(t, '<div class="product"><span>MIT component</span></div>');
  const sqlite = path.join(fixture.firefoxDirectory, 'libmozsqlite3.so');
  fs.writeFileSync(sqlite, 'synthetic SQLite binary');
  const environment = { ...process.env, FIREFOX_BINARY: fixture.firefoxExecutablePath };
  const identity = firefoxRuntimeIdentity(environment);
  const report = { schemaVersion: 2, browserVersion: 'test', ...identity };
  assert.equal(identity.firefoxSqliteSha256, fileDigest(sqlite));
  assert.equal(firefoxReportMatches(report, identity), true);
  fs.writeFileSync(sqlite, 'modified SQLite binary');
  assert.equal(firefoxReportMatches(report, firefoxRuntimeIdentity(environment)), false);
});

test('a matching target review is invalidated when a reviewed input changes', (t) => {
  const fixture = reviewedFixture(t, '<div class="product"><span>MIT component</span></div>');
  let result = fixture.run();
  assert.equal(result.status, hostIssues.length ? 1 : 0, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), hostIssues);

  const addedFile = path.join(fixture.electron, 'dist', 'new-binary');
  fs.writeFileSync(addedFile, 'changed distribution');
  result = fixture.run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), [...hostIssues, 'UNREVIEWED']);

  fs.unlinkSync(addedFile);
  fs.writeFileSync(path.join(fixture.directory, 'package-lock.json'), '{"changed":true}');
  result = fixture.run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), [...hostIssues, 'UNREVIEWED']);
});

test('Firefox runtime changes or missing smoke evidence block a matching review', (t) => {
  const fixture = reviewedFixture(t, '<div class="product"><span>MIT component</span></div>');
  let result = fixture.run();
  assert.equal(result.status, hostIssues.length ? 1 : 0, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), hostIssues);

  const resource = path.join(fixture.firefoxDirectory, 'new-runtime-component');
  fs.writeFileSync(resource, 'changed');
  result = fixture.run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), [
    ...hostIssues, 'UNREVIEWED', 'UNREVIEWED',
  ]);
  fs.unlinkSync(resource);

  const originalExecutable = fs.readFileSync(fixture.firefoxExecutablePath);
  fs.appendFileSync(fixture.firefoxExecutablePath, 'changed');
  result = fixture.run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), [
    ...hostIssues, 'UNREVIEWED', 'UNREVIEWED',
  ]);
  fs.writeFileSync(fixture.firefoxExecutablePath, originalExecutable);

  const report = path.join(fixture.directory, 'reports', 'firefox-runtime.json');
  fs.unlinkSync(report);
  result = fixture.run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), [...hostIssues, 'UNREVIEWED']);
});

test('Firefox launchers and unavailable runtimes fail closed', (t) => {
  const fixture = reviewedFixture(t, '<div class="product"><span>MIT component</span></div>');
  fs.writeFileSync(fixture.firefoxExecutablePath, '#!/bin/sh\nexit 0\n');
  const result = fixture.run();
  assert.equal(result.status, 1, result.stderr);
  const rejected = JSON.parse(result.stdout).rejected;
  assert.equal(rejected[hostIssues.length].license, 'UNREVIEWED');
  assert.match(rejected[hostIssues.length].reason, /native Firefox executable/);
  assert.equal(rejected.at(-1).license, 'UNREVIEWED');
});

test('a failed Firefox smoke removes stale runtime evidence', (t) => {
  const fixture = reviewedFixture(t, '<div class="product"><span>MIT component</span></div>');
  fs.writeFileSync(fixture.firefoxExecutablePath, 'not a native executable');
  const result = spawnSync(process.execPath, [path.join(__dirname, '..', 'scripts', 'smoke-firefox.cjs')], {
    cwd: fixture.directory,
    env: { ...process.env, FIREFOX_BINARY: fixture.firefoxExecutablePath },
    encoding: 'utf8',
  });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /native Firefox executable/);
  assert.equal(fs.existsSync(path.join(fixture.directory, 'reports', 'firefox-runtime.json')), false);
});

test('a matching binary audit cannot bypass the known FFT blocker', (t) => {
  const notices = '<div class="product">2-dim General Purpose FFT (Fast Fourier/Cosine/Sine Transform) Package<pre>You may distribute this ORIGINAL package.</pre></div>';
  const result = reviewedFixture(t, notices).run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), ['LicenseRef-scancode-ooura-2001', ...hostIssues]);
});

test('a matching binary audit cannot bypass the known JPEG blocker', (t) => {
  const notices = '<div class="product"><span class="title">libjpeg-turbo</span><pre>The IJG (Independent JPEG Group) License\nThis license applies to the libjpeg API library</pre></div>';
  const result = reviewedFixture(t, notices).run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), ['IJG', ...hostIssues]);
});

test('a matching binary audit cannot bypass the known SQLite blocker', (t) => {
  const notices = '<div class="product"><span class="title">sqlite</span><pre>The author disclaims copyright to this source code.</pre></div>';
  const result = reviewedFixture(t, notices).run();
  assert.equal(result.status, 1, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout).rejected.map((entry) => entry.license), ['blessing', ...hostIssues]);
});
