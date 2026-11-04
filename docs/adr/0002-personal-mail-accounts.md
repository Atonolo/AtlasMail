# ADR 0002: Personal IMAP/SMTP and JMAP accounts

**Status:** accepted for OpenSourceIdeas #35.  
**Context:** [account and sending feature](https://github.com/Atonolo/OpenSourceIdeas/issues/35) and [foundation contract](contracts-v1.md).

## Requirements

Support multiple independent personal accounts using IMAP/SMTP or JMAP from
both the standalone application and the self-hosted web surface. Verify TLS,
keep credentials out of configuration, exports and logs, preserve MIME bodies
and attachments, identify the sending account, and never retry an uncertain
send. Browser code must not connect directly to mail servers or load remote
message content.

## Decisions

- Implement IMAP, SMTP, JMAP, MIME parsing and MIME construction with the Go
  standard library. The adapter runs inside the existing Go service; the
  browser communicates only with the authenticated same-origin API. No new
  runtime or SDK dependency is introduced.
- Require implicit TLS or STARTTLS for IMAP and SMTP and verify the server
  certificate chain and hostname with the system trust store. JMAP uses HTTPS;
  HTTP is accepted only for loopback hosts to support local protocol tests.
  Certificate or STARTTLS failures stop the operation.
- Keep account identity in schema v1 and non-secret endpoint/authentication
  settings in a separately permission-restricted `connections-v1.json`.
  Passwords, app passwords, OAuth access tokens and refresh tokens exist only
  in the OS credential store. Connection settings and credentials are excluded
  from the existing state JSON export. Account removal revokes the keyring
  entry before deleting local connection and account data.
- Support password/app-password authentication and generic public OAuth 2.0
  Authorization Code with PKCE (S256). OAuth provider endpoints, public client
  ID and scopes are configured per account; client secrets and proprietary
  provider SDKs are not used. State and verifier are process-local, short-lived
  and single-use. The callback is bound to the exact local origin or configured
  HTTPS public origin. Refresh tokens rotate in the OS keyring; provider
  redirects are not followed.
- Load inbox messages only after an explicit user action. Parse text, HTML,
  common UTF-8/ISO-8859-1/Windows-1252 content, encoded headers and MIME
  attachments. The UI displays plain text (including a text-only extraction
  for HTML-only messages), never executes or inserts remote HTML, and offers
  attachments as local downloads. No tracking image or external resource is
  fetched.
- Require an explicit account selection for sending and use that account's
  address as the SMTP/JMAP sender. SMTP and JMAP submission errors after the
  delivery boundary return `SEND_STATUS_UNCERTAIN`; the application does not
  automatically retry.
- Bound messages, attachments, number of messages, MIME nesting and API
  payloads. Protocol errors are reduced to stable codes before they reach the
  UI or logs; credentials and provider response bodies are never logged.

## Consequences

- Profiles can be copied only by explicitly handling `connections-v1.json`;
  credentials must be re-entered or separately authorized by the OS keyring.
  The existing `state-v1.json` export remains free of provider settings and
  secrets.
- Public OAuth clients must support Authorization Code with PKCE and the
  registered callback URI. Providers requiring a confidential client secret
  or a proprietary authorization SDK are not supported by this generic flow.
- The initial inbox is fetched on demand with bounded message and attachment
  sizes; background synchronization, offline cache, rules, search, calendars
  and contacts remain separate features.
- A self-hosted instance is still a single-operator service. The operator's
  keyring stores the configured account credentials; multi-user identity and
  per-user credential isolation are not claimed.
