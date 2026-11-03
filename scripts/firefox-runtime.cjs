'use strict';

const fs = require('node:fs');
const path = require('node:path');
const { distributionDigest, fileDigest } = require('./digests.cjs');

function firefoxExecutable(environment = process.env, platform = process.platform) {
  if (environment.FIREFOX_BINARY) {
    if (!path.isAbsolute(environment.FIREFOX_BINARY)) throw new Error('FIREFOX_BINARY must be an absolute path');
    return environment.FIREFOX_BINARY;
  }
  if (platform === 'win32') {
    return path.join(environment.ProgramFiles || 'C:\\Program Files', 'Mozilla Firefox', 'firefox.exe');
  }
  if (platform === 'darwin') {
    return path.join(path.sep, 'Applications', 'Firefox.app', 'Contents', 'MacOS', 'firefox');
  }
  return 'firefox';
}

function resolveExecutable(executable, environment) {
  if (path.isAbsolute(executable)) return fs.realpathSync(executable);
  for (const directory of (environment.PATH || '').split(path.delimiter)) {
    if (!directory) continue;
    const candidate = path.join(directory, executable);
    try {
      fs.accessSync(candidate, fs.constants.X_OK);
      if (fs.statSync(candidate).isFile()) return fs.realpathSync(candidate);
    } catch (error) {
      if (error.code !== 'ENOENT' && error.code !== 'EACCES' && error.code !== 'ENOTDIR') throw error;
    }
  }
  throw new Error(`Firefox executable not found on PATH: ${executable}`);
}

function requireNativeExecutable(filename) {
  const descriptor = fs.openSync(filename, 'r');
  const header = Buffer.alloc(4);
  let length;
  try {
    length = fs.readSync(descriptor, header, 0, header.length, 0);
  } finally {
    fs.closeSync(descriptor);
  }
  const magic = header.toString('hex');
  const native = length === 4 && (
    (process.platform === 'win32' && magic.startsWith('4d5a')) ||
    (process.platform === 'linux' && magic === '7f454c46') ||
    (process.platform === 'darwin' && [
      'feedface', 'cefaedfe', 'feedfacf', 'cffaedfe', 'cafebabe', 'bebafeca', 'cafebabf', 'bfbafeca',
    ].includes(magic))
  );
  if (!native) {
    throw new Error('Select the native Firefox executable, not a launcher script, using FIREFOX_BINARY');
  }
}

function optionalFileDigest(filename) {
  try {
    return fileDigest(filename);
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
    return null;
  }
}

function firefoxRuntimeIdentity(environment = process.env) {
  const executable = resolveExecutable(firefoxExecutable(environment), environment);
  requireNativeExecutable(executable);
  const directory = path.dirname(executable);
  const coreName = { win32: 'xul.dll', darwin: 'XUL', linux: 'libxul.so' }[process.platform];
  const nssName = { win32: 'nss3.dll', darwin: 'libnss3.dylib', linux: 'libnss3.so' }[process.platform];
  if (!coreName || !nssName) throw new Error(`Unsupported Firefox platform: ${process.platform}`);
  const core = path.join(directory, coreName);
  let distribution = directory;
  if (process.platform === 'darwin' && path.basename(directory) === 'MacOS' &&
      path.basename(path.dirname(directory)) === 'Contents') {
    distribution = path.dirname(path.dirname(directory));
  }
  return {
    platform: process.platform,
    arch: process.arch,
    executable,
    firefoxExecutableSha256: fileDigest(executable),
    firefoxCoreSha256: fileDigest(core),
    firefoxNssSha256: optionalFileDigest(path.join(directory, nssName)),
    firefoxSqliteSha256: process.platform === 'linux' ? optionalFileDigest(path.join(directory, 'libmozsqlite3.so')) : null,
    firefoxDistributionSha256: distributionDigest(distribution),
  };
}

function firefoxReportMatches(report, identity) {
  if (!report || report.schemaVersion !== 2 || typeof report.browserVersion !== 'string' ||
      !report.browserVersion.trim() || !identity || typeof identity !== 'object') return false;
  const keys = [
    'platform', 'arch', 'executable', 'firefoxExecutableSha256',
    'firefoxCoreSha256', 'firefoxNssSha256', 'firefoxSqliteSha256', 'firefoxDistributionSha256',
  ];
  const digest = /^[a-f0-9]{64}$/;
  if (!['platform', 'arch', 'executable'].every((key) => typeof identity[key] === 'string' && identity[key]) ||
      !['firefoxExecutableSha256', 'firefoxCoreSha256', 'firefoxDistributionSha256']
        .every((key) => digest.test(identity[key])) ||
      !['firefoxNssSha256', 'firefoxSqliteSha256']
        .every((key) => identity[key] === null || digest.test(identity[key]))) return false;
  return keys.every((key) => report[key] === identity[key]);
}

function firefoxReportPath() {
  return path.resolve('reports', 'firefox-runtime.json');
}

function firefoxArguments(profile, platform = process.platform) {
  const args = ['-headless', '-no-remote'];
  if (platform === 'win32') args.push('-wait-for-browser');
  args.push('-profile', profile, '--remote-debugging-port=0');
  return args;
}

module.exports = {
  firefoxArguments, firefoxExecutable, firefoxReportMatches, firefoxReportPath, firefoxRuntimeIdentity,
};
