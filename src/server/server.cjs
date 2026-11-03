'use strict';

const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { ModelError } = require('../core/model.cjs');
const { StoreError } = require('../core/store.cjs');

const MAX_BODY_BYTES = 16 * 1024;
const UI_DIRECTORY = path.join(__dirname, '..', 'ui');
const STATIC_FILES = new Map([
  ['/', ['index.html', 'text/html; charset=utf-8']],
  ['/index.html', ['index.html', 'text/html; charset=utf-8']],
  ['/app.js', ['app.js', 'text/javascript; charset=utf-8']],
  ['/styles.css', ['styles.css', 'text/css; charset=utf-8']],
  ['/favicon.svg', ['favicon.svg', 'image/svg+xml']],
]);
const CSP = "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'";

function validToken(token) {
  return typeof token === 'string' && /^[A-Za-z0-9_-]{43,128}$/.test(token);
}

function authenticate(header, token) {
  if (typeof header !== 'string' || !header.startsWith('Bearer ')) return false;
  const actual = Buffer.from(header.slice(7), 'utf8');
  const expected = Buffer.from(token, 'utf8');
  return actual.length === expected.length && crypto.timingSafeEqual(actual, expected);
}

function response(res, status, data, contentType = 'application/json; charset=utf-8') {
  res.writeHead(status, {
    'Content-Type': contentType,
    'Cache-Control': 'no-store',
    'Content-Security-Policy': CSP,
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'Referrer-Policy': 'no-referrer',
    'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
  });
  res.end(contentType.startsWith('application/json') ? JSON.stringify(data) : data);
}

async function readJson(req) {
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/i.test(req.headers['content-type'] || '')) {
    const error = new Error('INVALID_CONTENT_TYPE');
    error.code = 'INVALID_CONTENT_TYPE';
    throw error;
  }
  return new Promise((resolve, reject) => {
    let length = 0;
    let rejected = false;
    const chunks = [];
    req.on('data', (chunk) => {
      if (rejected) return;
      length += chunk.length;
      if (length > MAX_BODY_BYTES) {
        rejected = true;
        const error = new Error('REQUEST_TOO_LARGE');
        error.code = 'REQUEST_TOO_LARGE';
        reject(error);
      } else {
        chunks.push(chunk);
      }
    });
    req.on('end', () => {
      if (rejected) return;
      try {
        resolve(JSON.parse(Buffer.concat(chunks).toString('utf8')));
      } catch (cause) {
        const error = new Error('INVALID_JSON', { cause });
        error.code = 'INVALID_JSON';
        reject(error);
      }
    });
    req.on('error', reject);
  });
}

function publicOrigin(value) {
  if (!value) return null;
  let url;
  try {
    url = new URL(value);
  } catch (cause) {
    throw new Error('ATLAS_MAIL_PUBLIC_ORIGIN must be an HTTPS origin', { cause });
  }
  if (
    url.protocol !== 'https:' ||
    url.username || url.password || url.pathname !== '/' ||
    url.search || url.hash
  ) {
    throw new Error('ATLAS_MAIL_PUBLIC_ORIGIN must be an HTTPS origin');
  }
  return url.origin;
}

function createServer({ service, token, publicOrigin: configuredOrigin }) {
  if (!validToken(token)) {
    throw new Error('ATLAS_MAIL_WEB_TOKEN must be a random base64url token of 43-128 characters');
  }
  const externalOrigin = publicOrigin(configuredOrigin);
  const server = http.createServer(async (req, res) => {
    try {
      const address = server.address();
      const port = address && typeof address === 'object' ? address.port : 0;
      const origins = externalOrigin
        ? [externalOrigin]
        : [`http://127.0.0.1:${port}`, `http://localhost:${port}`].map((origin) => new URL(origin).origin);
      const matchingOrigin = origins.find((origin) => new URL(origin).host === req.headers.host);
      if (!matchingOrigin || (req.headers.origin && req.headers.origin !== matchingOrigin)) {
        response(res, 403, { ok: false, code: 'ORIGIN_REJECTED' });
        return;
      }

      const requestUrl = new URL(req.url, matchingOrigin);
      if (req.method === 'GET' && STATIC_FILES.has(requestUrl.pathname)) {
        const [filename, contentType] = STATIC_FILES.get(requestUrl.pathname);
        const contents = fs.readFileSync(path.join(UI_DIRECTORY, filename));
        response(res, 200, contents, contentType);
        return;
      }
      if (!requestUrl.pathname.startsWith('/api/v1/')) {
        response(res, 404, { ok: false, code: 'NOT_FOUND' });
        return;
      }
      if (!authenticate(req.headers.authorization, token)) {
        response(res, 401, { ok: false, code: 'AUTH_REQUIRED' });
        return;
      }
      if (req.method === 'GET' && requestUrl.pathname === '/api/v1/status') {
        response(res, 200, { ok: true, data: service.status() });
        return;
      }
      if (req.method === 'POST' && requestUrl.pathname === '/api/v1/accounts') {
        const account = service.addAccount(await readJson(req));
        response(res, 201, { ok: true, data: account });
        return;
      }
      response(res, 404, { ok: false, code: 'NOT_FOUND' });
    } catch (error) {
      if (error instanceof ModelError) {
        response(res, 400, { ok: false, code: error.code });
      } else if (['INVALID_CONTENT_TYPE', 'INVALID_JSON', 'REQUEST_TOO_LARGE'].includes(error.code)) {
        response(res, error.code === 'REQUEST_TOO_LARGE' ? 413 : 400, { ok: false, code: error.code });
      } else if (error instanceof StoreError) {
        console.error(`Atlas Mail storage error: ${error.code}`);
        response(res, error.code === 'STORE_LOCKED' ? 409 : 500, { ok: false, code: error.code });
      } else {
        console.error('Atlas Mail request failed.');
        response(res, 500, { ok: false, code: 'UNEXPECTED_ERROR' });
      }
    }
  });
  return server;
}

module.exports = { createServer, validToken };
