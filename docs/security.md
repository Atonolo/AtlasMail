# Security and trust boundaries

The P0 handles only synthetic configuration. Trust boundaries are the
sandboxed desktop renderer/main-process IPC, the browser/loopback HTTP API,
the filesystem, and the OS credential service. A local operator is the only
P0 role. Separate household users, provider identities and authorization
policies are not implemented. A malicious local process running as the same
user is not fully isolated by this design.

The desktop disables Node integration, enables sandbox/context isolation,
restricts IPC to its main frame, blocks navigation and window creation, and
blocks non-file requests in the desktop session. It refuses to open when
`--no-sandbox` is present. This flag check is not proof that the OS sandbox
is functioning. Linux AppImage packaging was disabled because the examined
electron-builder launcher silently supplied that flag when user namespaces
were restricted; the supported Ubuntu package is DEB. Background networking,
component updates and sync are disabled through Chromium switches; a
packet-level outbound-traffic audit is still pending. UI data is inserted as
text. The web API accepts
only a configured bearer token, checks Host and Origin, provides no CORS
access and is loopback-only. A remote deployment requires a correctly
configured TLS reverse proxy; no TLS termination is implemented in the Node
service itself. Tokens are held only in browser memory and must not enter logs,
URLs, local storage, backups or source control. The operator must generate a
high-entropy token; length/character checks alone cannot prove entropy.

Electron `safeStorage` encrypts credential blobs with Windows DPAPI, macOS
Keychain or a supported Linux keyring. Linux's `basic_text` fallback is
rejected. Windows DPAPI does not protect against all other programs running
as the same user; unsigned macOS builds may trigger repeated Keychain
prompts. The synchronous safeStorage API used here is deprecated in Electron
45 and must be replaced with a verified, fail-closed asynchronous provider
before upgrading to Electron 46. The async Linux API has its own fallback
providers, so the synchronous backend name alone must not be assumed to prove
which async key provider was selected. No credential entry or web credential
backend is exposed in P0.

Before [Atonolo/OpenSourceIdeas#35](https://github.com/Atonolo/OpenSourceIdeas/issues/35)
is enabled, provider OAuth must use the authorization-code flow with PKCE and
validated redirects/state, with tokens only in a protected backend; server
deployments need a separately designed secret manager and user/session
authorization. HTML messages are untrusted: the future renderer must sanitize
content, isolate it in a sandboxed origin/frame, block remote images and
scripts by default, and treat links/attachments as untrusted. No HTML message
is rendered by P0. No plugin or telemetry feature is implemented.

Errors are surfaced with non-secret codes. Storage is user-private by
directory permissions where supported; Windows ACLs and OS account security
remain the operator's responsibility. Backups of configuration are not
encrypted automatically and exclude OS-bound credentials. See
[contracts-v1.md](contracts-v1.md) for failure and recovery limits.
