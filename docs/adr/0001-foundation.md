# ADR 0001: Local desktop core and self-hosted web boundary

**Status:** proposed foundation; platform evidence and license resolution pending.
**Context:** [Atonolo/OpenSourceIdeas#5](https://github.com/Atonolo/OpenSourceIdeas/issues/5)
and [Atonolo/OpenSourceIdeas#1](https://github.com/Atonolo/OpenSourceIdeas/issues/1).

## Decision and alternatives

The primary form is an installable standalone application. A separately
self-hostable web surface is also mandatory, but a browser cannot speak IMAP
directly. Electron 45.2.1 (MIT) provides one desktop UI across platforms;
the same HTML/CSS/JavaScript is served to the browser. Node.js 26.13.0 is the
development/CI runtime and built-in Node APIs provide the shared model,
transactional file store and HTTP boundary. electron-builder 26.18.0 (MIT)
is configured for baseline installers. Built-in `node:test` provides tests.
The project uses MPL-2.0. Direct versions were checked against npm metadata
and npm resolutions are locked. Electron's bundled notices list Ooura FFT,
IJG-derived libjpeg-turbo code and public-domain SQLite, none under an
OSI-approved license. The official symbols for all four target binaries
confirm that these components are linked, so this runtime choice is **not
approved for release**. They must be removed or replaced in accordance with
the issue's license requirement. The remaining bundled license evidence is
also incomplete. The Firefox browser smoke adds a test runtime: matched
Windows and Linux Firefox ESR 153.6.0 and Windows Firefox 159.0.1 symbols
confirm IJG-derived JPEG and public-domain SQLite there as well. These
versions also fail the issue's OSI-only requirement and are not an approved
release test toolchain.

The current Linux installer target is DEB. The initially selected AppImage
target was removed after its generated launcher automatically passed
`--no-sandbox` on the tested Ubuntu configuration. The application rejects
that flag; a future portable package must preserve isolation and be verified
before it is reintroduced.

Tauri/Rust would reduce bundled size but introduces another toolchain and a
second runtime boundary; Qt would require a separate browser UI. Flutter would
require a different UI/toolchain, and its examined Windows release engine also
links IJG-derived code. An isolated wxWidgets Windows sample started with
several optional components disabled but used Microsoft build and runtime
tools; a replacement UI, runtime and their licenses remain unverified. See
[the license evidence](../../THIRD_PARTY_NOTICES.md). None of these alternatives
has a complete cross-platform license audit.
A browser-only PWA would not satisfy the standalone installer requirement.
JSON was chosen over SQLite for this small initial synthetic configuration,
with strict v1 validation and atomic same-directory replacement. A later
cache/database migration must preserve v1 data and add a tested, recoverable
migration.

```text
Desktop UI ---- sandboxed preload ----+
                                     +-- MailService v1 -- StateStore v1
Browser UI -- authenticated HTTP ----+       |                 |
  HTTPS reverse proxy (optional)             |                 +-- state-v1.json
                                             +-- protocol adapter contracts
Desktop main -- OS safeStorage -- encrypted credential files
```

The desktop works without the web service. The optional web service runs from
the same codebase on loopback; a TLS reverse proxy can expose it. The web
surface does not access the desktop credential store and no protocol adapter
is implemented. A single operator bearer token authorizes the P0 web API;
it is not a family account system. The browser has no service worker or
offline cache: it requires the local/self-hosted service, although neither
surface requires Internet access for the synthetic-data walkthrough.

## Target matrix

| Target | Intended baseline | Evidence |
|---|---|---|
| Windows | Windows 11, x64 | Local smoke and unsigned NSIS install, restart, uninstall and data-retention checks passed on one host; clean-VM confirmation pending |
| macOS | macOS 14+, Intel and Apple Silicon | Both CI runners configured; execution, signing and installation pending |
| Linux | Ubuntu 24.04+, x64, graphical desktop and Secret Service/KWallet | Desktop, web and GNOME-keyring smoke passed in an isolated Ubuntu VM; final package and CI evidence pending |
| Web | Current Chrome/Edge, Firefox ESR and Safari 17+ on those OSes | Windows browser checks and Ubuntu Firefox ESR smoke passed; browser/OS matrix pending |

These are intended support targets, not claims of verified compatibility.
Windows 11 and Ubuntu 24.04 LTS are current mainstream desktop baselines;
macOS 14+ is the intended Apple baseline, with Intel and Apple Silicon both
retained for existing users. The CI macOS 15 images do not themselves prove
macOS 14 compatibility. Mobile remains a later additional target. The npm
lockfile pins dependency resolutions; CI runner images and separately
downloaded binaries can change. See [verification](../verification.md).

## Ownership and integration

`src/core` owns the internal v1 model, storage, service and protocol contracts.
`src/ui` is shared by the standalone and web surfaces. `src/desktop` owns
Electron isolation and its OS-backed credential adapter; `src/server` owns
the HTTP boundary. No separate OS codebases exist. Until a suite-wide shared
package is agreed, Atlas Mail owns these contracts and does not claim that a
shared design system or library has already been published.
The current light theme is provisional; the suite-wide design system and dark
mode remain open work and are not represented as delivered.

Notes/Mail task references and Mail/Meet event references use the stable UUIDs
specified in [contracts-v1.md](../contracts-v1.md); compatibility, rights,
offline and idempotency tests are prerequisites of the relevant integration
features. Mail-to-Write is a future read-only adapter, not direct access to
the local store. No integration is silently claimed to exist.

## Consequences and follow-up gates

- [#35](https://github.com/Atonolo/OpenSourceIdeas/issues/35) must implement
  authenticated IMAP/SMTP/JMAP and the production web protocol adapter,
  provider OAuth and secure server-side key handling before real credentials
  or message content can be used.
- [#38](https://github.com/Atonolo/OpenSourceIdeas/issues/38) must implement
  cache payloads, durable queue execution, offline behavior and conflict
  handling before anything is marked offline-complete.
- [#37](https://github.com/Atonolo/OpenSourceIdeas/issues/37) owns actual
  CalDAV/CardDAV, calendars, contacts and task synchronization.
- [#40](https://github.com/Atonolo/OpenSourceIdeas/issues/40) owns OpenPGP and
  its key lifecycle. [#36](https://github.com/Atonolo/OpenSourceIdeas/issues/36)
  owns Meet/Drive integration. No PGP, PST import, or integrations are claimed.
- Web multi-user authentication, the future optional family calendar service,
  HTML message rendering policy implementation, signing/update channels, and
  shared suite UI ownership require decisions before their respective features.

Windows and scoped Linux VM evidence is recorded in
[verification](../verification.md). The full cross-platform matrix, macOS
package installation, accessibility audit and complete dependency audit
remain pending.
