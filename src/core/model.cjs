'use strict';

const SCHEMA_VERSION = 1;
const MAX_RECORDS = 10000;
const IDENTIFIER = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

class ModelError extends Error {
  constructor(code) {
    super(code);
    this.name = 'ModelError';
    this.code = code;
  }
}

function assertObject(value, fields) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new ModelError('INVALID_DATA');
  }
  const keys = Object.keys(value);
  if (keys.length !== fields.length || fields.some((field) => !Object.hasOwn(value, field))) {
    throw new ModelError('INVALID_DATA');
  }
}

function assertIdentifier(value) {
  if (typeof value !== 'string' || !IDENTIFIER.test(value)) {
    throw new ModelError('INVALID_IDENTIFIER');
  }
}

function assertText(value, maxLength, allowEmpty = false) {
  if (
    typeof value !== 'string' ||
    value.length > maxLength ||
    (!allowEmpty && (!value.length || value.trim() !== value))
  ) {
    throw new ModelError('INVALID_DATA');
  }
}

function assertChoice(value, choices) {
  if (!choices.includes(value)) {
    throw new ModelError('INVALID_DATA');
  }
}

function emptyState() {
  return {
    schemaVersion: SCHEMA_VERSION,
    accounts: [],
    messages: [],
    calendars: [],
    events: [],
    contacts: [],
    tasks: [],
    cache: [],
    queue: [],
  };
}

function makeAccount(input, id) {
  assertObject(input, ['label', 'address', 'protocol']);
  assertIdentifier(id);
  if (typeof input.label !== 'string' || typeof input.address !== 'string') {
    throw new ModelError('INVALID_ACCOUNT');
  }
  const account = {
    id,
    label: input.label.trim(),
    address: input.address.trim(),
    protocol: input.protocol,
  };
  validateAccount(account);
  return account;
}

function validateAccount(account) {
  assertObject(account, ['id', 'label', 'address', 'protocol']);
  assertIdentifier(account.id);
  assertText(account.label, 80);
  assertText(account.address, 254);
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(account.address)) {
    throw new ModelError('INVALID_ACCOUNT');
  }
  assertChoice(account.protocol, ['imap-smtp', 'jmap']);
}

function validateState(state) {
  if (!state || typeof state !== 'object' || Array.isArray(state)) {
    throw new ModelError('INVALID_DATA');
  }
  if (state.schemaVersion !== SCHEMA_VERSION) {
    throw new ModelError('UNSUPPORTED_SCHEMA');
  }
  const collections = [
    'accounts', 'messages', 'calendars', 'events',
    'contacts', 'tasks', 'cache', 'queue',
  ];
  assertObject(state, ['schemaVersion', ...collections]);
  for (const name of collections) {
    if (!Array.isArray(state[name]) || state[name].length > MAX_RECORDS) {
      throw new ModelError('INVALID_DATA');
    }
  }

  const identifiers = new Set();
  const accounts = new Set();
  const calendars = new Map();
  const entities = new Map();
  const register = (id) => {
    assertIdentifier(id);
    if (identifiers.has(id)) throw new ModelError('DUPLICATE_IDENTIFIER');
    identifiers.add(id);
  };
  const requireAccount = (id) => {
    assertIdentifier(id);
    if (!accounts.has(id)) throw new ModelError('INVALID_REFERENCE');
  };
  const registerEntity = (record, kind, owner) => {
    register(record.id);
    entities.set(record.id, { kind, owner });
  };

  for (const account of state.accounts) {
    validateAccount(account);
    register(account.id);
    accounts.add(account.id);
  }
  for (const message of state.messages) {
    assertObject(message, ['id', 'accountId', 'remoteId', 'folderId']);
    requireAccount(message.accountId);
    assertText(message.remoteId, 1024, true);
    assertText(message.folderId, 1024, true);
    registerEntity(message, 'message', message.accountId);
  }
  for (const calendar of state.calendars) {
    assertObject(calendar, ['id', 'accountId', 'remoteId']);
    requireAccount(calendar.accountId);
    assertText(calendar.remoteId, 1024, true);
    register(calendar.id);
    calendars.set(calendar.id, calendar.accountId);
  }
  for (const event of state.events) {
    assertObject(event, ['id', 'calendarId', 'remoteId']);
    assertIdentifier(event.calendarId);
    if (!calendars.has(event.calendarId)) throw new ModelError('INVALID_REFERENCE');
    assertText(event.remoteId, 1024, true);
    registerEntity(event, 'event', calendars.get(event.calendarId));
  }
  for (const kind of ['contacts', 'tasks']) {
    for (const entry of state[kind]) {
      assertObject(entry, ['id', 'accountId', 'remoteId']);
      requireAccount(entry.accountId);
      assertText(entry.remoteId, 1024, true);
      registerEntity(entry, kind === 'contacts' ? 'contact' : 'task', entry.accountId);
    }
  }
  for (const kind of ['cache', 'queue']) {
    for (const entry of state[kind]) {
      const fields = kind === 'cache'
        ? ['id', 'accountId', 'entityId', 'kind', 'state']
        : ['id', 'accountId', 'entityId', 'kind', 'status', 'createdAt'];
      assertObject(entry, fields);
      register(entry.id);
      requireAccount(entry.accountId);
      assertIdentifier(entry.entityId);
      assertChoice(entry.kind, ['message', 'event', 'contact', 'task']);
      const target = entities.get(entry.entityId);
      if (!target || target.owner !== entry.accountId || target.kind !== entry.kind) {
        throw new ModelError('INVALID_REFERENCE');
      }
      if (kind === 'cache') {
        assertChoice(entry.state, ['metadata']);
      } else {
        assertChoice(entry.status, ['pending', 'in-progress', 'failed', 'complete']);
        if (
          typeof entry.createdAt !== 'string' ||
          !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(entry.createdAt) ||
          Number.isNaN(Date.parse(entry.createdAt)) ||
          new Date(entry.createdAt).toISOString() !== entry.createdAt
        ) {
          throw new ModelError('INVALID_DATA');
        }
      }
    }
  }
  return state;
}

module.exports = {
  SCHEMA_VERSION,
  ModelError,
  assertIdentifier,
  emptyState,
  makeAccount,
  validateState,
};
