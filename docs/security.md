# Security and trust boundaries

P0 stores synthetic configuration only. The boundaries are the operating
system browser, the loopback HTTP service, the filesystem and the native OS
credential store. There is one local operator; multi-user authorization,
provider identities and household permissions are not implemented. A
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
Do not share or log launch URLs. Same-user process inspection is outside
the stated boundary. Skip-link fragments are not interpreted as session tokens.

Both modes apply exact Host/listening-port and Origin checks, a restrictive
content security policy, no CORS, and response hardening headers.
Self-hosted mode binds to
loopback, requires a high-entropy bearer token and accepts an exact HTTPS
public origin only when configured behind a reverse proxy. The service itself
does not terminate TLS. Server bearer tokens are held only in UI memory,
never in URLs or localStorage, and must not be logged by a proxy.

Credential operations use the OS keyring through `go-keyring`: Windows
Credential Manager, macOS Keychain or Linux Secret Service. Missing keyring
services fail closed. Secrets are limited to 2048 bytes for cross-platform
backend compatibility. `credentials-smoke` verifies native write/read/delete
using an isolated synthetic identifier; errors are explicit, without secrets.
Credentials are not present in the state file, backups,
browser UI or API. No credential-entry or protocol operation is enabled in
this foundation. Before account integration, define OAuth PKCE, token
rotation/revocation, server-side key custody and authorization.

The UI inserts account values as text. It renders no message HTML and loads no
remote resources. Future message display must sanitize untrusted HTML, isolate
it in a sandboxed origin/frame, block remote images and scripts by default,
and treat links and attachments as untrusted. Telemetry and background
network requests are not implemented.

State files and temporary writes use restrictive permissions where supported.
Backups may contain personal configuration and are not encrypted; OS-bound
credentials are excluded. Windows ACL and local OS account security remain
the operator's responsibility.
