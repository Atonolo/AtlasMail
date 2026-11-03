'use strict';

class FeatureUnavailableError extends Error {
  constructor(feature) {
    super(`FEATURE_UNAVAILABLE: ${feature}`);
    this.name = 'FeatureUnavailableError';
    this.code = 'FEATURE_UNAVAILABLE';
    this.feature = feature;
  }
}

class UnavailableAdapter {
  constructor(protocol) {
    this.protocol = protocol;
  }

  capabilities() {
    return { contractVersion: 1, protocol: this.protocol, available: false };
  }

  async connect() {
    throw new FeatureUnavailableError(this.protocol);
  }

  async synchronize() {
    throw new FeatureUnavailableError(this.protocol);
  }

  async send() {
    throw new FeatureUnavailableError(this.protocol);
  }
}

function createAdapters() {
  return Object.fromEntries(
    ['imap', 'smtp', 'jmap', 'caldav', 'carddav'].map(
      (protocol) => [protocol, new UnavailableAdapter(protocol)],
    ),
  );
}

module.exports = { FeatureUnavailableError, createAdapters };
