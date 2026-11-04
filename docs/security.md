# Security and trust boundaries

Personal IMAP/SMTP and JMAP credentials are used by the Go service, never by
browser-side protocol clients. The boundaries are the operating system
browser, authenticated HTTP service, filesystem, remote mail provider and
native OS credential store. There is one local operator; multi-user
authorization and per-user credential isolation are not implemented. A
malicious process running as the same user is outside the isolation boundary.

Standalone mode binds an ephemeral port to `127.0.0.1`, generates a random
32-byte process token, and opens `/desktop#session=<token>`. JavaScript removes
the fragment from history immediately and bootstraps a nonpersistent HttpOnly,
SameSite=Strict cookie. Subsequent local API requests require that cookie and
`X-Atlas-Local: 1`, or the explicit bearer token. Cookies are named per port;
browser cookie transport itself is not port-isolated. Restarting the process
invalidates earlier session tokens. The local HTTP cookie is not marked
Secure because this listener intentionally uses loopback HTTP.

The launch fragment is briefly visible to the browser launcher, process
arguments and initial history, but is not sent in HTTP URLs or referrers.
Do not share or log launch URLs. The UI stores only the non-sensitive theme
preference in localStorage; session tokens are never stored there. Same-user
process inspection is outside the stated boundary. Skip-link fragments are
not interpreted as session tokens.

Both modes apply exact Host/listening-port and Origin checks, a restrictive
content security policy, no CORS, and response hardening headers.
Self-hosted mode binds to
loopback, requires a high-entropy bearer token and accepts an exact HTTPS
public origin only when configured behind a reverse proxy. The service itself
does not terminate TLS. Server bearer tokens are held only in UI memory,
never in URLs or localStorage, and must not be logged by a proxy.

Credential operations use the OS keyring through `go-keyring`: Windows
Credential Manager, macOS Keychain or Linux Secret Service. Missing keyring
services fail closed. Passwords and tokens are limited to 2048 bytes for
cross-platform backend compatibility. `credentials-smoke` verifies native
write/read/delete using an isolated synthetic identifier. Credentials are
never written into state/profile files, included in backups, returned by APIs,
or included in logs/errors. The password travels from the authenticated UI to
the service only during account setup; OAuth tokens stay server-side.
Removing an account revokes its keyring entry before deleting local metadata.

IMAP/SMTP use implicit TLS or STARTTLS with system roots and hostname
verification; invalid certificates and failed STARTTLS abort the operation.
JMAP requires HTTPS except loopback HTTP used for local tests. JMAP session,
upload and download URLs must remain same-origin, and redirects are disabled.
OAuth Authorization Code uses PKCE S256, short-lived single-use state, a
configured public client ID and HTTPS authorization/token endpoints. Token
redirects are not followed. Tokens are refreshed only through the configured
provider endpoint and updated in the keyring.

The UI inserts account values and message fields as text. It does not render
untrusted message HTML or fetch remote resources; HTML-only bodies are reduced
to text. Attachments are offered as local downloads. Message and attachment
sizes, nesting and counts are bounded. Sending requires an explicit account;
after the delivery boundary, uncertain SMTP/JMAP status is reported and never
automatically retried. Telemetry and background network requests are not
implemented.

State files and temporary writes use restrictive permissions where supported.
Backups may contain personal configuration and are not encrypted; OS-bound
credentials are excluded. Windows ACL and local OS account security remain
the operator's responsibility.
