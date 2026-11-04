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
go mod download all
go test ./...
go vet ./...
go run .
```

The desktop command starts an ephemeral loopback listener and opens
`http://127.0.0.1:<port>/desktop#session=<one-time-process-token>` in the system
browser. The UI immediately removes the fragment and exchanges the token for
an HttpOnly session cookie; API calls remain authenticated after reloading.
The token changes on every application restart. Do not copy the launch URL
into logs or share it. Stop the foreground
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

## Local release gate, packages and SBOM

```sh
go run ./cmd/verify -race -all-platforms
```

This runs uncached contract/restart tests, static analysis, dependency
integrity/license checks, SBOM/notices generation, and six cross-builds.
The race detector requires a supported host and C compiler; omit `-race`
when unavailable and record that limitation. Omit `-all-platforms` to build
only the current host. Evidence is written to ignored `dist/local/`:
`verification.log`, `verification.json` (revision, host, toolchain, SHA-256s),
`licenses.json`, `sbom.cdx.json`, `DEPENDENCY-LICENSES.txt`, and portable ZIPs.
Unreviewed dependencies or licenses stop packaging with a nonzero exit.

Extract the matching ZIP into a user-owned application directory and run
`AtlasMail/atlas-mail` (`AtlasMail/atlas-mail.exe` on Windows). No Go installation
is needed to run it. Stop the app before replacing the entire extracted folder
for an update. Uninstall by deleting that folder; user data remains at the
separate location above. To remove personal data too, stop all instances and
explicitly delete that specific AtlasMail data directory. These are portable
baseline packages, not signed native installers or an automatic updater.

Individual audit commands are also available:

```sh
go run ./cmd/licenseaudit
go run ./cmd/licenseaudit -sbom
go run ./cmd/licenseaudit -notices
```

The dependency graph is pinned by `go.mod` and `go.sum`. The blocking license
audit verifies downloaded contents, rejects replacements, checks exact
versions and SHA-256 fingerprints of reviewed original license texts, and
includes Go's runtime and build-tool vendor inventories. The application and
direct/transitive Go modules use OSI-approved licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). The audit includes the
pinned Go standard library/toolchain and original redistribution notices.
No Chromium, Node.js,
SQLite, browser runtime, or third-party
installer framework is bundled. The desktop uses the browser supplied by the
user's operating system. This does not constitute a license approval of that
external browser or every system/build/test component; their artifact-specific
reviews remain required before full acceptance.

## Data safety and limitations

Schema v1 rejects unknown/missing/case-aliased/duplicate fields, null values,
unsupported versions, duplicate identifiers and invalid references. Writes validate before replacing the previous file,
use a same-directory temporary file, and take an exclusive lock. Backups
contain configuration only; credentials are kept separately by the native OS
credential store and are not yet exposed in the P0 interface. See
[security boundaries](docs/security.md) before using a self-hosted deployment.
If a process crashes and leaves a `state-v1.json.lock` file, stop every Atlas
Mail process before removing that specific lock file. Never remove it while
another instance may be writing.

The existing GitHub Actions workflow is unchanged and describes the retired
Electron/npm foundation. Do not run it: the issue explicitly requires local
evidence instead of GitHub workflows or status checks. Windows and macOS package,
credential-store, and full assistive-technology checks also require those
operating systems and are not claimed based on cross-compilation.

## Deutsche Kurz-Anleitung und Fehlerdiagnose

Voraussetzungen: Go 1.27.3 zum Bauen, ein Systembrowser und fuer native
Zugangsdaten der Betriebssystem-Schluesselspeicher. Unter Linux muss eine
entsperrte Secret-Service-Sitzung laufen. Die Oberflaeche bietet Deutsch und
Englisch; echte Kontenanmeldung und Synchronisierung sind noch nicht verfuegbar.

Frischer Einstieg: Repository auschecken, `go mod download all`,
`go test ./...`, `go vet ./...` und `go run .` ausfuehren. Der lokale Kern
benoetigt weder Herstellerkonto noch Cloudservice. Fuer lokale Abnahme und
portable Pakete `go run ./cmd/verify -race -all-platforms` ausfuehren.
Passendes ZIP entpacken und `atlas-mail` beziehungsweise `atlas-mail.exe`
starten. Aktualisierung: Prozess stoppen und Programmordner ersetzen.
Deinstallation: nur Programmordner loeschen; persoenliche Daten bleiben
in den oben genannten separaten Verzeichnissen erhalten.

Datensicherung und Datenuebernahme: alle Instanzen stoppen, `state-v1.json`
kopieren und bei gestoppter Anwendung am Ziel ablegen. Schema v1 wird vor
Verwendung vollstaendig validiert; unbekannte Versionen nicht umschreiben.
Credentials werden dabei nicht kopiert und muessen spaeter im jeweiligen
OS-Schluesselspeicher neu eingerichtet werden. Es gibt keine automatische
Migration, keinen Mailimport und noch keine Familienfreigaben.

`AUTH_REQUIRED`: Server-Token erneut eingeben; im lokalen Modus die Anwendung
neu starten und deren frisch geoeffnete Browserseite verwenden.
`CONNECTION_FAILED`: laufenden Prozess und Loopback-Adresse pruefen; ungespeicherte
Formulare bleiben bei Fehlern erhalten. `STORE_LOCKED`: andere Instanzen
beenden; eine verwaiste `state-v1.json.lock` nur entfernen, wenn garantiert
kein Schreibprozess mehr laeuft. `INVALID_DATA` oder `UNSUPPORTED_SCHEMA`:
Originaldatei sichern und nicht durch einen leeren Datenbestand ersetzen.
`KEYRING_UNAVAILABLE`: nativen Schluesselspeicher und entsperrte Benutzersitzung
pruefen; `atlas-mail credentials-smoke` prueft mit synthetischen Daten
Schreiben, Lesen und Loeschen. Niemals echte Zugangsdaten in Fehlermeldungen
oder Berichte uebernehmen.

English troubleshooting: `AUTH_REQUIRED` needs a fresh server token or local
application restart; `CONNECTION_FAILED` means the service is unreachable.
For `STORE_LOCKED`, stop other instances before considering stale-lock removal.
Keep the original file on validation/schema errors. `credentials-smoke`
checks the real OS keyring using synthetic data, not a plaintext fallback.
Backup/restore is a stopped-process copy of `state-v1.json`; credentials are
excluded. Family sharing and automatic migrations are not available.

## License

The [application source](https://github.com/Atonolo/AtlasMail) is licensed
under the Mozilla Public License 2.0; see
[LICENSE](LICENSE). Contributions are covered by
[CONTRIBUTING.md](CONTRIBUTING.md).
