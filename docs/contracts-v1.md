# Atlas Mail contracts, version 1

`src/core/model.cjs` is the authoritative validator and
`test/fixtures/state-v1.json` is the synthetic reference case. A state document
has exactly `schemaVersion: 1` and the arrays `accounts`, `messages`,
`calendars`, `events`, `contacts`, `tasks`, `cache`, and `queue`. Unknown
versions and fields are rejected, not silently discarded. Each collection is
limited to 10,000 records and the serialized document to 10 MiB.

All `id` values are globally unique, canonical lowercase UUIDs. An account has
`id`, `label`, `address` and `protocol` (`imap-smtp` or `jmap`); it contains
no password, token or key. Messages have `id`, `accountId`, `remoteId`,
`folderId`; calendars have `id`, `accountId`, `remoteId`; events refer to a
calendar by `calendarId`; contacts and tasks refer to an account by
`accountId`. Protocol identifiers are opaque and may be empty for local
objects. They are not substitutes for stable local IDs.

Cache entries reference an existing entity and its owning account and have
`id`, `accountId`, `entityId`, `kind`, and `state`. Version 1 accepts only
`metadata`: no payload is stored and nothing is asserted to be available
offline. A future complete state requires verified payloads and a versioned
migration. Queue records have the same identity/reference fields, a `status`
(`pending`, `in-progress`, `failed`, `complete`), and an ISO UTC `createdAt`.
They are data structures only; nothing is sent, retried or synchronized.
Cross-account or wrong-kind references are rejected.

`MailService.status()` returns `contractVersion: 1`, `schemaVersion`, accounts,
adapter capabilities and the credential-store availability boundary.
`MailService.addAccount({label,address,protocol})` generates a local UUID and
persists the synthetic configuration. IPC and HTTP wrap results as
`{ok:true,data}` or `{ok:false,code}`. The HTTP endpoints are
`GET /api/v1/status` and `POST /api/v1/accounts`, require a bearer token, and
never accept credentials. Future API changes require an explicit version.

The IMAP, SMTP, JMAP, CalDAV and CardDAV adapters expose v1 capability records
with `available:false`; `connect`, `synchronize` and `send` reject with
`FEATURE_UNAVAILABLE`. They do not simulate successful protocol work.

The desktop credential store takes a local account UUID and uses Electron's
OS-backed `safeStorage`. A missing keyring, Linux `basic_text`, or an unknown
Linux backend fails closed. It is not exposed to the renderer or web API.
Future consumers must avoid logging secrets and must specify revocation,
rotation, backup/recovery and web-host key custody before enabling credentials.

The core writes a same-directory temporary file, flushes it and renames it
over the current state while holding an exclusive lock. Validation and any
failure before the rename leave the old file untouched. A restore validates
the entire backup before replacing a valid current state. Unknown schemas
are rejected without attempting a migration. Power-loss durability, shared
network filesystems, full concurrent-process recovery and cross-device
credential recovery are not guaranteed by this foundation. If lock cleanup
fails after an operation started, `COMMIT_UNCERTAIN` explicitly means that the
write may already have succeeded; callers must inspect the state before
retrying.
