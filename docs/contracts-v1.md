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

- `GET /api/v1/status` returns the contract/schema versions, synthetic
  accounts, explicit unavailable adapter capabilities and credential-store
  boundary.
- `POST /api/v1/accounts` accepts only `{label,address,protocol}` and returns
  the generated stable account ID.
- Local mode only: `POST /api/v1/session` requires the launch bearer token
  and issues a browser-session cookie scoped to `/api/v1/`.

Responses have `{ok:true,data}` or `{ok:false,code}`. Self-hosted mode requires
a high-entropy bearer token; standalone mode requires its random per-process
token or a bootstrap cookie plus `X-Atlas-Local: 1`. Both modes validate
Host against the actual listening port and reject mismatched Origin headers
(or require the configured exact HTTPS public origin). Neither mode accepts
provider credentials. IMAP, SMTP, JMAP, CalDAV and
CardDAV remain unavailable instead of reporting simulated success.

Configuration is saved as `state-v1.json`, separate from keyring-managed
credentials. The store uses a same-directory temporary file, file sync,
atomic replacement, restrictive file permissions and an exclusive lock.
Backups are validated before restore. Unknown schemas and failed validation
do not overwrite the last valid file. Shared network filesystems and
power-loss guarantees are not claimed. A failed lock cleanup after writing
reports `COMMIT_UNCERTAIN`; inspect the state before retrying.
