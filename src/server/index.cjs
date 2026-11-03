'use strict';

const { StateStore } = require('../core/store.cjs');
const { dataDirectory } = require('../core/paths.cjs');
const { MailService } = require('../core/service.cjs');
const { createServer } = require('./server.cjs');

function main() {
  const port = Number(process.env.ATLAS_MAIL_PORT || 3123);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error('ATLAS_MAIL_PORT must be an integer from 1 to 65535');
  }
  const server = createServer({
    service: new MailService(new StateStore(dataDirectory())),
    token: process.env.ATLAS_MAIL_WEB_TOKEN,
    publicOrigin: process.env.ATLAS_MAIL_PUBLIC_ORIGIN,
  });
  server.on('error', (error) => {
    console.error(`Atlas Mail service could not start: ${error.code || 'UNKNOWN_ERROR'}`);
    process.exitCode = 1;
  });
  server.listen(port, '127.0.0.1', () => {
    console.log(`Atlas Mail web foundation listening on 127.0.0.1:${port}`);
  });
}

try {
  main();
} catch (error) {
  console.error(`Atlas Mail configuration error: ${error.message}`);
  process.exitCode = 1;
}
