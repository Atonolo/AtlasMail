# Atlas Mail contracts, version 1

`internal/model/state.go` is the authoritative validator. A state document
contains exactly `schemaVersion: 1` and the arrays `accounts`, `messages`,
`calendars`, `events`, `contacts`, `tasks`, `cache`, and `queue`. Unknown
versions, fields, missing fields, nulls, duplicate/case-aliased keys, trailing
JSON, invalid raw UTF-8 and unpaired UTF-16 escapes are rejected rather than
silently replaced. Each
collection is limited to 10,000 records and the serialized document to 10 MiB.

All IDs are globally unique lowercase UUIDs. An account has `id`, `label`,
`address` and `protocol` (`imap-smtp` or `jmap`) and never contains a password,
token or key. Messages, calendars, events, contacts and tasks carry stable IDs
and references. Cache records currently permit only `metadata`; they do not
claim that content is available offline. Queue records are data only and are
not sent, retried or synchronized. Cross-account and wrong-kind references
fail validation.
Text limits count UTF-16 code units to preserve the original v1 contract.
`internal/model/testdata/state-v1.json` is the complete versioned reference
fixture, including metadata cache and structural queue records.

The shared API is versioned at `/api/v1`:

- `GET /api/v1/status` returns the contract/schema versions, configured
  account identities and adapter/credential-store status. It never returns
  credentials.
- `POST /api/v1/accounts` accepts only `{label,address,protocol}` and returns
  the generated stable account ID. This metadata-only route does not configure
  a working mail connection.
- `POST /api/v1/accounts/setup` accepts `{account,profile,secret}`. `account`
  has `label`, `address`, and `protocol`; `profile` has `username`,
  `authMechanism`, and either `imap`/`smtp` endpoints or a `jmapUrl`. Endpoints
  use `host`, `port`, and `tlsMode` (`implicit` or `starttls`). Passwords and
  app passwords are passed as `secret`; OAuth profiles instead provide
  `oauth.authorizationUrl`, `oauth.tokenUrl`, `oauth.clientId`, and `oauth.scope`.
  The secret is written only to the OS credential store and is never echoed.
- `DELETE /api/v1/accounts/{id}` removes the keyring credential, connection
  profile and account record.
- `GET /api/v1/mail?accountId={id}` explicitly fetches a bounded set of inbox
  messages. Message bodies and attachment data are returned only to this
  authenticated client.
- `POST /api/v1/mail/send` accepts a `mail.SendInput` with an explicit
  `accountId`, recipient list, text/HTML alternatives and attachments. A
  conflict `SEND_STATUS_UNCERTAIN` means delivery may have occurred; the
  client must not retry automatically.
- `GET /api/v1/oauth/start?accountId={id}` begins a short-lived PKCE flow and
  returns the provider authorization URL. The callback is handled at
  `/oauth/callback`; credentials remain server-side.
- Local mode only: `POST /api/v1/session` requires the launch bearer token
  and issues a browser-session cookie scoped to `/api/v1/`.

Responses have `{ok:true,data}` or `{ok:false,code}`. Self-hosted mode requires
a high-entropy bearer token; standalone mode requires its random per-process
token or a bootstrap cookie plus `X-Atlas-Local: 1`. Both modes validate
Host against the actual listening port and reject mismatched Origin headers
(or require the configured exact HTTPS public origin). Neither mode accepts
The browser never connects directly to mail servers. IMAP/SMTP use verified
implicit TLS or STARTTLS. JMAP requires HTTPS except for loopback-only HTTP
test endpoints. CalDAV and CardDAV remain unavailable instead of reporting
simulated success.

Account identities are saved in `state-v1.json`; non-secret connection
profiles are saved in the separate, restrictive `connections-v1.json`.
Neither file contains passwords or OAuth tokens. Secrets are stored in the
native OS credential store and are excluded from state exports/backups.
`state-v1.json` uses a same-directory temporary file, file sync, atomic
replacement, restrictive file permissions and an exclusive lock. Backups are
validated before restore. Unknown schemas and failed validation do not
overwrite the last valid file. Shared network filesystems and power-loss
guarantees are not claimed. A failed lock cleanup after writing reports
`COMMIT_UNCERTAIN`; inspect the state before retrying.
