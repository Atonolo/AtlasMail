# Verification and acceptance evidence

Issue [Atonolo/OpenSourceIdeas#5](https://github.com/Atonolo/OpenSourceIdeas/issues/5)
requires local evidence, not GitHub workflows/status checks. The legacy
Electron workflow is unchanged and must not be run. Portable packages and
cross-builds do not establish native installation or full issue acceptance.

## Reproducible local gate

With Go 1.27.3 and a race-supported host/C compiler:

```sh
go run ./cmd/verify -race -all-platforms
```

The command downloads the pinned source graph, runs uncached tests and vet,
verifies module integrity, exact versions and reviewed license fingerprints,
emits CycloneDX 1.5 and original redistribution notices, and builds CGO-free
portable ZIPs for Windows/macOS/Linux on amd64 and arm64. An unknown component,
replacement or license mismatch exits nonzero before packaging. The old
success summary is removed before a new run. Omit `-race` on unsupported
hosts and explicitly record the missing race check.

Ignored `dist/local/` contains `verification.log`, `verification.json`,
`licenses.json`, `sbom.cdx.json`, `DEPENDENCY-LICENSES.txt` and six ZIPs.
The summary identifies the source revision, dirty-worktree flag, Go/host
versions, artifact SHA-256s and acceptance items not established by this gate.
Rerun on the committed source for a clean, revision-specific evidence set.
ZIPs include executable permission, README, LICENSE, third-party notices,
original dependency licenses, SBOM and dependency locks.

## Automated coverage

Tests cover the complete versioned fixture, exact required/case-sensitive
JSON fields, null/duplicate/trailing input, invalid UTF-8, UTF-16 limits,
stable IDs and cross-account references. Persistence coverage includes
backup/restore, concurrent locks, injected replacement permission failure,
temporary-file cleanup, and preserving unsupported-schema/last-valid bytes.
Invalid escaped UTF-16 and invalid in-memory UTF-8 cannot be silently replaced
during decoding or persistence.
An actual subprocess is terminated and restarted against the same directory;
the account identity and attributes survive. The HTTP tests check bearer
authentication, per-process local cookie bootstrap/reload, mandatory custom
local header, Host port, Origin rejection and visible failures.

Mail coverage uses local test servers and synthetic credentials only. It
checks IMAP/SMTP TLS and certificate rejection, MIME parsing/construction and
attachments, JMAP receive/send and uncertain submission, OAuth PKCE state,
credential separation/removal, account API behavior and visible protocol
errors. It does not prove interoperability with any real mail provider or
provider-specific OAuth settings.

## Scoped Linux manual evidence, 2026-11-04

Host: Linux amd64 development container; Go 1.27.3. These are diagnostic
checks, not a clean Ubuntu desktop or a complete dependency/license approval.

| Check | Artifact/tool | Result |
|---|---|---|
| Real browser form submission | HeadlessChrome 156.0.0.0, shared embedded UI | Enter submits one metadata-only synthetic account; count/address visible |
| Actual service stop and failed submit | Server binary SHA-256 `d82e73b53af9a0f5ef9e9d85b7298644a9aa608f0e5a5a3f60b28f50794d79f8` | Visible `CONNECTION_FAILED`; no success status; unsaved form retained |
| Actual service restart | Same binary and data directory | Saved account count remains 1 with same address; failed offline submission absent |
| Keyboard/localization diagnostic | Same browser | Space activates German; Tab advances to English with solid 3px focus outline |
| Browser storage/network diagnostic | Same browser | Token field cleared; no tokens in localStorage (theme preference only); observed resource origins loopback-only |
| Portable desktop authentication | Linux ZIP executable SHA-256 `4f080a0da2a2f370eccb0fc3ac040e16627a431fabb8a70ea707ed0d7ebdd2d3` | Captured native-launcher URL opened in real browser; fragment removed, HttpOnly cookie hidden from JavaScript; saved account survives reload including `#main`; cookie-only call without local header returns 401 |
| Portable removal/reinstallation | Linux ZIP executable SHA-256 `8b8a134ab6a925bd4acd7b4e6d86717be3c774a3bb55db6303005e9b9c3c729e` | Removed only extracted program directory, re-extracted ZIP, started server mode; retained desktop account returned by authenticated API |
| Final portable native keyring smoke | Same `8b8a134a...` executable | Fresh unprivileged profile/D-Bus session; real native write/read/delete passed |
| Local gate and archive inspection | Go 1.27.3 Linux amd64, all six targets | Race tests, vet, license gate, builds, SHA-256s, ZIP notices/docs and executable bits passed |
| SBOM standards check | Official CycloneDX 1.5/SPDX schemas, Python jsonschema 4.26.0 (isolated diagnostic tool) | Schema validation passed; shared runtime/build-vendor entries deduplicated; conflicting reviews fail |
| Native Secret Service write/read/delete | GNOME keyring `46.1-2ubuntu0.2`, libsecret `0.21.4-1build3`, dbus-x11 `1.14.10-4ubuntu4.1` | `credentials-smoke` passed in isolated unprivileged user profile/D-Bus session |

The native-keyring executable SHA-256 was
`719aea27ada9259745c72ee10c4bf537dac7eba917003d48e75c858f238eae56`.
These diagnostic binaries were built from the uncommitted continuation of
`91471a4`; none is represented as a clean final release artifact. The final
revision-specific hashes are generated by rerunning the local gate on the
committed tree. Native smoke first failed
explicitly without Secret Service, then passed after installing/unlocking
the isolated test service. No real credentials were used.

## Acceptance still open

- Actual Windows 11/macOS 14+ execution, keyring integration and browser
  start/persistence on Intel/Apple Silicon/arm64 targets.
- Native installer/signing/update behavior, and clean-host installation,
  launch/removal with data-retention checks on all three platforms.
- Chrome/Edge/Firefox/Safari target-version/OS matrix and full WCAG 2.1 AA,
  keyboard and screen-reader acceptance. One focus check is not that review.
- Artifact-complete legal review of external browser/runtime, system/native
  credential services, host compiler and test tooling. Top-level module and
  toolchain license fingerprints alone do not prove every bundled file.
- Provider-specific OAuth registration, live token exchange/refresh, and
  interoperability with real IMAP/SMTP/JMAP providers using isolated test
  accounts; generic OAuth and local protocol fakes do not establish this.
- Browser UI account setup, mailbox navigation, HTML-only composition, OAuth
  popup, and attachment flows on the target browser/OS matrix.

The Go graph uses reviewed OSI licenses and does not bundle Electron, Node,
SQLite or a browser. This removes those bundled-runtime blockers but does
not approve an external browser by omission. Unknown components still block
full acceptance/distribution. Do not mark the original issue fully accepted
or these missing native checks passed based on local cross-compilation.
