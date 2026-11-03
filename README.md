# Atlas Mail

Atlas Mail is a local-first personal information manager foundation for the
Atlas Office suite. Its primary application is a standalone Go executable:
starting it launches the shared accessible UI in the operating system's
browser, backed by an embedded loopback-only service. The same executable can
also run the separately authenticated, self-hosted web service.

The selected target is the already existing public repository
`Atonolo/AtlasMail`, checked on 2026-11-03; it was selected explicitly and
was not overwritten or recreated.

Only synthetic accounts are supported in this foundation. Mail, calendar,
contact, task, synchronization, OAuth and message rendering remain explicitly
unavailable. The initial storage schema and adapter contracts are versioned.
See [the architecture decision](docs/adr/0001-foundation.md),
[the data/API contract](docs/contracts-v1.md), and
[the verification record](docs/verification.md).

## Requirements and local development

- Go 1.27.3 (pinned by `go.mod`)
- Windows 11 x64/arm64, macOS 14+ Intel/Apple Silicon, or Ubuntu 24.04+ Linux
  x64/arm64 (the Go standard library is cross-platform)
- A system browser for desktop mode
- An OS credential store for credential operations: Secret Service on Linux,
  Keychain on macOS, or Windows Credential Manager. The P0 UI does not accept
  real credentials.

```sh
go test ./...
go vet ./...
go run .
```

The desktop command starts an ephemeral loopback listener and opens
`http://127.0.0.1:<port>/desktop` in the system browser. Stop the foreground
process with Ctrl+C. The application stores synthetic data in
`%APPDATA%\AtlasMail\state-v1.json`, `~/Library/Application Support/AtlasMail/state-v1.json`,
or `${XDG_DATA_HOME:-~/.local/share}/AtlasMail/state-v1.json`. Set
`ATLAS_MAIL_DATA_DIR` to an absolute path to select another directory.
The portable executable is installed by placing it in a user-owned
application directory; removing that executable does not remove personal
data. Native installers and signing are not provided yet.

On Linux, a user-local build can be installed with:

```sh
mkdir -p "$HOME/.local/bin"
go build -trimpath -o "$HOME/.local/bin/atlas-mail" .
```

## Self-hosted web service

Generate a random 32-byte base64url token and start the same executable:

```sh
export ATLAS_MAIL_WEB_TOKEN="$(go run ./cmd/token)"
go run . server
```

The service binds only to `127.0.0.1:3123` by default. `ATLAS_MAIL_PORT`
selects another local port. For remote use, put a self-managed HTTPS reverse
proxy in front of the loopback service and set `ATLAS_MAIL_PUBLIC_ORIGIN` to
the exact HTTPS origin. The proxy must preserve the `Host` header and must not
log authorization headers. Browser API requests require the configured
bearer token; it is held in memory only. Do not expose the Go listener
directly to a network.

## Build, dependency audit and SBOM

```sh
go build -trimpath -o dist/atlas-mail .
GOOS=windows GOARCH=amd64 go build -trimpath -o dist/atlas-mail.exe .
GOOS=darwin GOARCH=arm64 go build -trimpath -o dist/atlas-mail .
GOOS=linux GOARCH=amd64 go build -trimpath -o dist/atlas-mail .
go run ./cmd/licenseaudit
go run ./cmd/licenseaudit -sbom > reports/sbom.cdx.json
```

The dependency graph is pinned by `go.mod` and `go.sum`. The blocking license
audit checks every module in Go's module graph against its reviewed SPDX
license and verifies the corresponding license text. The application and
direct/transitive Go modules use OSI-approved licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). The audit includes the
pinned Go standard library/toolchain (BSD-3-Clause). No Chromium, Node.js,
SQLite, browser runtime, or third-party
installer framework is bundled. The desktop uses the browser supplied by the
user's operating system.

## Data safety and limitations

Schema v1 rejects unknown fields, unsupported versions, duplicate identifiers
and invalid references. Writes validate before replacing the previous file,
use a same-directory temporary file, and take an exclusive lock. Backups
contain configuration only; credentials are kept separately by the native OS
credential store and are not yet exposed in the P0 interface. See
[security boundaries](docs/security.md) before using a self-hosted deployment.
If a process crashes and leaves a `state-v1.json.lock` file, stop every Atlas
Mail process before removing that specific lock file. Never remove it while
another instance may be writing.

The existing GitHub Actions workflow was intentionally left unchanged as
requested; it still describes the retired Electron/npm foundation and is not
a validation path for this Go implementation. Run the Go commands above
locally until workflow changes are authorized. Windows and macOS package,
credential-store, and full assistive-technology checks also require those
operating systems and are not claimed based on cross-compilation.

## License

The application source is licensed under the Mozilla Public License 2.0; see
[LICENSE](LICENSE). Contributions are covered by
[CONTRIBUTING.md](CONTRIBUTING.md).
