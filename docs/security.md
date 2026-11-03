# Security and trust boundaries

P0 stores synthetic configuration only. The boundaries are the operating
system browser, the loopback HTTP service, the filesystem and the native OS
credential store. There is one local operator; multi-user authorization,
provider identities and household permissions are not implemented. A
malicious process running as the same user is outside the isolation boundary.

Standalone mode binds an ephemeral port to `127.0.0.1`, opens a local route,
and applies strict Host and Origin checks, a restrictive content security
policy, no CORS, and response hardening headers. Self-hosted mode binds to
loopback, requires a high-entropy bearer token and accepts an exact HTTPS
public origin only when configured behind a reverse proxy. The service itself
does not terminate TLS. Authorization tokens are not placed in URLs or local
storage and must not be logged by a proxy.

Credential operations use the OS keyring through `go-keyring`: Windows
Credential Manager, macOS Keychain or Linux Secret Service. Missing keyring
services fail closed. Credentials are not present in the state file, backups,
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
