# ADR 0001: Go executable with a shared local web UI

**Status:** accepted for the P0 redevelopment; target operating-system and
accessibility acceptance remains subject to native testing.
**Context:** [Atonolo/OpenSourceIdeas#5](https://github.com/Atonolo/OpenSourceIdeas/issues/5)
and [Atonolo/OpenSourceIdeas#1](https://github.com/Atonolo/OpenSourceIdeas/issues/1).

## Requirements

The foundation must provide a local standalone application and a separately
self-hostable web surface from one codebase. Both need the same accessible UI,
versioned model, transactional persistence and honest unavailable states.
Runtime, build and test dependencies must use OSI-approved licenses. The web
surface must not suggest direct browser-to-IMAP connectivity. Credentials
must use the OS keyring, not plaintext or the configuration file.

## Decision

The product is implemented in Go 1.27.3 using the standard library for the
process entry point, HTTP service, JSON contracts, persistence, random UUIDs,
and tests. The UI remains plain HTML, CSS and JavaScript and is embedded into
the executable with `embed`. Standalone mode starts an ephemeral loopback
service and opens `/desktop` in the user's system browser. Server mode starts
the same service with bearer-token authentication and a configurable local
port. No desktop shell, browser engine, database, external service, or
installer framework is bundled.

`github.com/zalando/go-keyring` 0.2.6 (MIT) is the only direct Go module. Its
locked transitive modules use MIT, BSD-2-Clause, BSD-3-Clause or Apache-2.0;
the exact module list and license evidence are maintained in
[THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md). The Go toolchain and
standard library are BSD-3-Clause. The application remains MPL-2.0. A
source-controlled audit allowlist blocks unreviewed module paths and missing
or mismatched license texts; an SBOM can be emitted from the same locked
module graph.

The previously selected Electron/Chromium and Node runtime has been removed.
Its bundled non-OSI FFT, IJG JPEG and public-domain SQLite components could
not meet the product's OSI-only dependency requirement. The npm/electron
builder chain has also been removed.

## Alternatives

- **Electron/Chromium/Node:** rejected because matched runtime binaries contain
  linked non-OSI components and Node bundles SQLite.
- **Tauri, Qt, Flutter, wxWidgets and Gio:** not adopted because the examined
  candidates either retained licensing/accessibility uncertainties or
  required a second, unverified UI and deployment stack.
- **Native per-platform clients:** rejected because they would create
  separate product implementations and duplicate the required shared UI.
- **Browser-only PWA:** rejected because it would not provide the required
  installable standalone application.
- **Go plus OS browser:** selected because it keeps the app and web service in
  one portable executable, shares the real UI, and avoids bundling a browser.

## Component boundaries

```text
Standalone executable -- loopback HTTP -- embedded shared UI
                            |-- MailService v1
                            |-- StateStore v1 -- state-v1.json
                            |-- protocol contracts (unavailable)
                            `-- OS credential-store adapter

Self-hosted executable -- authenticated loopback HTTP -- same components
```

The configuration model contains no secret material. The native credential
adapter uses Windows Credential Manager, macOS Keychain or Linux Secret
Service through the keyring library. The P0 UI deliberately does not collect
or send credentials. IMAP, SMTP, JMAP, CalDAV, CardDAV, message HTML rendering,
sync and durable queue execution remain future features.

## Platform contract and evidence

The same source tree builds for Windows, macOS and Linux. Intended baselines
are Windows 11 x64/arm64, macOS 14+ Intel/Apple Silicon, and Ubuntu 24.04+
Linux x64/arm64. The browser is supplied by the operating system; supported
browser families are current Chrome/Edge and Firefox, plus Safari 17+ on
macOS. No browser version is bundled or implied. Cross-compilation is a build
check, not an execution, installer, keyring or accessibility test. Native
package, browser matrix and assistive-technology evidence remains tracked in
[verification.md](../verification.md).

## Consequences

- `go.mod` and `go.sum` are the dependency manifest and lock.
- `go test ./...` covers contracts, persistence, API authentication and error
  paths; the module/license audit and CycloneDX output use `cmd/licenseaudit`.
- Desktop mode has no vendor login and no required network or cloud service.
- The standalone app relies on the system browser and the foreground
  executable process; stopping the process stops the app.
- The existing Actions workflow is unchanged per request and still targets
  the retired npm/Electron implementation. It must not be treated as evidence
  for this ADR until workflow changes are authorized.
- Signing, update channels, platform-specific package installers, complete
  WCAG review and the user-facing credential flow remain open work.
