'use strict';

const os = require('node:os');
const path = require('node:path');

function dataDirectory(environment = process.env, platform = process.platform) {
  if (environment.ATLAS_MAIL_DATA_DIR) {
    if (!path.isAbsolute(environment.ATLAS_MAIL_DATA_DIR)) {
      throw new Error('ATLAS_MAIL_DATA_DIR must be an absolute path');
    }
    return environment.ATLAS_MAIL_DATA_DIR;
  }
  const home = os.homedir();
  if (platform === 'win32') {
    const appData = environment.APPDATA || path.join(home, 'AppData', 'Roaming');
    if (!path.isAbsolute(appData)) throw new Error('APPDATA must be an absolute path');
    return path.join(appData, 'AtlasMail');
  }
  if (platform === 'darwin') {
    return path.join(home, 'Library', 'Application Support', 'AtlasMail');
  }
  const dataHome = environment.XDG_DATA_HOME;
  if (dataHome && !path.isAbsolute(dataHome)) throw new Error('XDG_DATA_HOME must be an absolute path');
  return path.join(dataHome || path.join(home, '.local', 'share'), 'AtlasMail');
}

module.exports = { dataDirectory };
