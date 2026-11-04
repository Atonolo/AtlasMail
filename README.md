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

## Schritt fuer Schritt: Windows und macOS

Diese Anleitung installiert die **portable Entwicklungsversion**, keinen
signierten Installer. Die native Abnahme steht noch aus; die folgenden
Schritte sind die Anleitung fuer diese manuelle Pruefung, kein Nachweis ihres
Bestehens. Verwenden Sie ausschliesslich synthetische Testkonten.

Die ZIPs unter `dist/local/` werden lokal erzeugt und sind nicht im Repository
enthalten. Ein GitHub-Quellcode-ZIP ist kein ausfuehrbares Programmpaket.
Der hier beschriebene Weg baut deshalb zuerst aus dem Quellcode. Dafuer ist
ein Internetzugang zum Herunterladen von Werkzeugen und Dependencies
erforderlich; der anschliessende lokale Betrieb braucht keinen Cloudservice.
Es werden keine GitHub-Workflows verwendet.

### Windows 11 (x64 oder ARM64)

1. Unter **Einstellungen > System > Info > Systemtyp** pruefen, ob ein
   x64- oder ARM64-System verwendet wird. Einen Standardbrowser einrichten.
   [Git fuer Windows](https://git-scm.com/downloads/win) und
   [Go 1.27.3](https://go.dev/dl/) fuer die passende Architektur installieren.
   Danach eine neue **PowerShell** oeffnen, nicht als Administrator.

2. Werkzeuge pruefen und den Implementierungsbranch herunterladen:

   ```powershell
   git --version
   go version
   git clone --branch atonolo-upgraded-couscous --single-branch https://github.com/Atonolo/AtlasMail.git
   Set-Location AtlasMail
   git rev-parse HEAD
   go env GOOS GOARCH
   ```

   `go version` muss `go1.27.3 windows/amd64` beziehungsweise
   `go1.27.3 windows/arm64` anzeigen. Die Commit-ID fuer den Testbericht notieren.
   Nicht zum Default-Branch wechseln: Dieser kann einen anderen Stand enthalten.

3. Lokale Pruefung und Paketbau starten:

   ```powershell
   $env:CGO_ENABLED = "0"
   go run ./cmd/verify
   ```

   Nur bei Exitcode `0` fortfahren (`$LASTEXITCODE` pruefen). Der Befehl erzeugt
   Tests, Lizenzbericht, SBOM und ein ZIP fuer die Architektur der Go-Installation
   in `dist\local`. Er benoetigt keinen C-Compiler. Der optionale Race-Test ist
   hierbei **nicht** enthalten; diese Einschraenkung im Bericht festhalten.
   Schlaegt die Lizenzpruefung fehl, nicht umgehen oder ein ungeprueftes Paket
   verwenden.

4. Paket pruefen und in das eigene Benutzerprofil installieren:

   ```powershell
   $arch = (go env GOARCH).Trim()
   $zip = Join-Path (Get-Location) "dist\local\atlas-mail-windows-$arch.zip"
   $report = Get-Content .\dist\local\verification.json -Raw | ConvertFrom-Json
   $expected = ($report.artifacts | Where-Object { $_.file -eq "atlas-mail-windows-$arch.zip" }).sha256
   if (-not $expected -or (Get-FileHash $zip -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) {
       throw "Paketpruefsumme fehlt oder stimmt nicht."
   }
   $installRoot = Join-Path $env:LOCALAPPDATA "Programs\AtlasMailPortable"
   if (Test-Path $installRoot) {
       throw "Ziel existiert bereits. Zuerst die Hinweise zur Aktualisierung beachten."
   }
   Expand-Archive -LiteralPath $zip -DestinationPath $installRoot
   $exe = Join-Path $installRoot "AtlasMail\atlas-mail.exe"
   ```

   Der Pruefsummenvergleich erkennt eine Abweichung vom lokalen Baubericht,
   ersetzt aber keine Signatur oder Vertrauenspruefung der Quelle.

5. In derselben PowerShell starten:

   ```powershell
   & $exe
   ```

   Der Standardbrowser sollte Atlas Mail oeffnen. Die PowerShell bleibt
   belegt, solange der Dienst laeuft. Mit **Strg+C** beenden; nur das Schliessen
   des Browser-Tabs beendet ihn nicht. Fuer spaetere Starts:

   ```powershell
   & "$env:LOCALAPPDATA\Programs\AtlasMailPortable\AtlasMail\atlas-mail.exe"
   ```

   Windows-Warnungen nicht durch Abschalten von SmartScreen oder Virenschutz
   umgehen. Bei einer Blockierung Quelle, Commit und Pruefsumme kontrollieren
   und die genaue Meldung als fehlgeschlagenen Test dokumentieren.
   Eine Netzwerkfreigabe fuer oeffentliche/private Netze ist fuer den
   Loopback-Betrieb nicht erforderlich.

6. Nach dem Beenden die native Zugangsdatenverwaltung pruefen:

   ```powershell
   & "$env:LOCALAPPDATA\Programs\AtlasMailPortable\AtlasMail\atlas-mail.exe" credentials-smoke
   ```

   Erwartet werden Exitcode `0` und
   `Native credential-store write/read/delete smoke passed (synthetic data).`
   Der Test verwendet Windows Credential Manager mit einem zufaelligen
   synthetischen Eintrag und entfernt diesen wieder.

7. Daten liegen getrennt vom Programm unter
   `%APPDATA%\AtlasMail\state-v1.json`. Zur Sicherung alle Instanzen beenden
   und diese Datei an einen sicheren Ort kopieren. Zur Wiederherstellung
   bei gestoppter Anwendung die Datei an genau diesen Ort zurueckkopieren;
   vorhandene Daten vorher separat sichern.

### macOS 14 oder neuer (Apple Silicon oder Intel)

1. Unter **Apple-Menue > Ueber diesen Mac** den Chip pruefen: Apple Silicon
   verwendet `arm64`, Intel `amd64`. Einen Standardbrowser einrichten.
   [Go 1.27.3](https://go.dev/dl/) als passendes macOS-Paket installieren.
   Git wird ebenfalls benoetigt; beispielsweise
   [Git installieren](https://git-scm.com/downloads/mac).
   Falls Apple beim Aufruf von Git die Command Line Tools anbietet, diese
   bewusst installieren. Danach ein neues **Terminal** oeffnen.

2. Werkzeuge pruefen und den Implementierungsbranch herunterladen:

   ```sh
   git --version
   go version
   git clone --branch atonolo-upgraded-couscous --single-branch https://github.com/Atonolo/AtlasMail.git
   cd AtlasMail
   git rev-parse HEAD
   go env GOOS GOARCH
   ```

   Erwartet wird `go1.27.3 darwin/arm64` beziehungsweise
   `go1.27.3 darwin/amd64`. Die Commit-ID notieren. Fuer native Apple-Silicon-
   Tests keine Intel-Go-Installation unter Rosetta verwenden.

3. Tests, Lizenzpruefung und das native portable Paket erzeugen:

   ```sh
   CGO_ENABLED=0 go run ./cmd/verify
   ```

   Nur bei erfolgreichem Abschluss fortfahren. Das ZIP und die Berichte
   liegen unter `dist/local/`. Dieser Weg benoetigt keinen C-Compiler und
   fuehrt keinen Race-Test aus; das im Testbericht vermerken. Mit verfuegbarem
   C-Compiler kann separat `go run ./cmd/verify -race` verwendet werden.
   Lizenzfehler nicht umgehen.

4. Paketpruefsumme mit `verification.json` vergleichen und installieren:

   ```sh
   arch="$(go env GOARCH)"
   shasum -a 256 "dist/local/atlas-mail-darwin-$arch.zip"
   cat dist/local/verification.json
   ```

   Der ausgegebene Hash muss exakt zum Eintrag
   `atlas-mail-darwin-arm64.zip` beziehungsweise `atlas-mail-darwin-amd64.zip`
   passen. Erst danach in ein noch nicht vorhandenes Benutzerverzeichnis
   entpacken:

   ```sh
   (
     set -eu
     destination="$HOME/Applications/AtlasMailPortable"
     if [ -e "$destination" ]; then
       printf '%s\n' "Ziel existiert bereits; Aktualisierungshinweise beachten." >&2
       exit 1
     fi
     mkdir -p "$destination"
     unzip "dist/local/atlas-mail-darwin-$(go env GOARCH).zip" -d "$destination"
     chmod u+x "$destination/AtlasMail/atlas-mail"
   )
   ```

   Das Paket ist keine `.app` und kein notarisiertes `.dmg`. Die Pruefsumme
   vergleicht den lokalen Build, nicht eine Herstellersignatur.

5. Im Terminal starten, nicht per Finder-Doppelklick:

   ```sh
   "$HOME/Applications/AtlasMailPortable/AtlasMail/atlas-mail"
   ```

   Atlas Mail sollte im Standardbrowser erscheinen. Das Terminal offen
   lassen; **Ctrl+C** beendet den Prozess. Den Startbefehl fuer spaetere
   Starts erneut verwenden. Gatekeeper, SIP oder Quarantaeneschutz nicht
   pauschal deaktivieren. Bei einer Blockierung die Meldung dokumentieren
   und Quelle/Commit pruefen; eine solche Blockierung gilt nicht als
   bestandener Installationstest.

6. Nach dem Beenden die native Schluesselbund-Anbindung pruefen:

   ```sh
   "$HOME/Applications/AtlasMailPortable/AtlasMail/atlas-mail" credentials-smoke
   ```

   Die normale angemeldete Benutzersitzung und ein entsperrter
   Anmeldeschluesselbund werden benoetigt. Eventuelle Schluesselbund-Dialoge
   pruefen. Erwartet werden Exitcode `0` und die oben genannte
   `Native credential-store write/read/delete smoke passed`-Meldung.
   Keine echten Mail-Zugangsdaten eingeben.

7. Die Daten befinden sich unter
   `~/Library/Application Support/AtlasMail/state-v1.json`.
   Im Finder kann **Gehe zu > Gehe zum Ordner** diesen Pfad oeffnen.
   Zur Sicherung alle Instanzen stoppen und die Datei kopieren;
   zur Wiederherstellung die gesicherte Datei bei gestoppter Anwendung
   zuruecklegen, nachdem der bisherige Stand separat gesichert wurde.

### Manuelle Pruefung nach der Installation (beide Systeme)

Jeden Schritt mit **bestanden / fehlgeschlagen / nicht geprueft** protokollieren.
OS-Version, CPU, Browser samt Version, Commit-ID und Paket-SHA-256 sowie
`dist/local/verification.json` und `verification.log` aufbewahren.
Berichte duerfen keine Start-URL mit Sitzungstoken oder echten Zugangsdaten
enthalten. Ein bestandener Durchlauf auf einem System gilt nur fuer diese
konkrete Kombination, nicht fuer die gesamte Plattformmatrix.

1. Starten: Die lokale Oberflaeche erscheint ohne Herstellerkonto.
   Deutsch und Englisch umschalten; noch nicht implementierte Mail-/DAV-/
   Sync-Funktionen muessen als nicht verfuegbar erkennbar bleiben.
2. Nur mit **Tab**, **Umschalt+Tab**, **Enter** und **Leertaste** navigieren:
   Sprunglink, Sprachwahl und Formular erreichen; sichtbaren Fokus und
   sinnvolle Reihenfolge pruefen. Mit VoiceOver auf macOS beziehungsweise
   einem Screenreader auf Windows Beschriftungen, Status und Fehlermeldungen
   pruefen. Dies dokumentiert den Basisablauf, keine vollstaendige WCAG-Abnahme.
3. Ein Testkonto `Manuelle Abnahme` / `manual@example.invalid` speichern.
   Erfolg und Kontenzahl pruefen; Browser neu laden und denselben Eintrag
   erwarten. Mit Ctrl+C/Strg+C den Prozess stoppen, erneut starten und
   gespeicherte Werte wieder pruefen.
4. Bei geoeffneter Seite den Prozess stoppen und ein weiteres synthetisches
   Konto absenden: Eine sichtbare Verbindungsfehlermeldung statt Erfolg muss
   erscheinen, die Eingabe soll erhalten bleiben. Nach Neustart darf das
   fehlgeschlagene Konto nicht gespeichert sein. Die neu geoeffnete Seite
   benutzen, weil die alte Sitzung nach Neustart ungueltig ist.
5. `credentials-smoke` wie oben ausfuehren und Ergebnis dokumentieren.
6. Den Pflicht-Webdienst aus demselben Quellstand pruefen. Zuerst die lokale
   Anwendung beenden, dann im Terminal im Quellcode-Verzeichnis:

   Windows/PowerShell:

   ```powershell
   $env:ATLAS_MAIL_WEB_TOKEN = (go run ./cmd/token)
   Write-Host $env:ATLAS_MAIL_WEB_TOKEN
   & "$env:LOCALAPPDATA\Programs\AtlasMailPortable\AtlasMail\atlas-mail.exe" server
   ```

   macOS/Terminal:

   ```sh
   export ATLAS_MAIL_WEB_TOKEN="$(go run ./cmd/token)"
   printf '%s\n' "$ATLAS_MAIL_WEB_TOKEN"
   "$HOME/Applications/AtlasMailPortable/AtlasMail/atlas-mail" server
   ```

   Der Token wird hier nur zur lokalen Uebernahme angezeigt: nicht aufzeichnen,
   teilen oder in Screenshots/Testprotokolle kopieren. Nicht in einer
   Terminal-Sitzung mit laufender Protokollaufzeichnung ausfuehren.
   `http://127.0.0.1:3123` oeffnen, den Token im Anmeldeformular eingeben,
   ein synthetisches Konto speichern, Dienst stoppen und mit demselben
   Startbefehl samt `server` im selben Terminal neu starten. Wieder anmelden
   und Persistenz pruefen. Ohne Token darf kein Zugriff auf Kontodaten moeglich
   sein. Die Webdienstpruefung ist fuer die volle Abnahme erforderlich,
   auch wenn sie fuer einen ersten Installationsversuch ausgelassen wird.
7. Aktualisierung/Deinstallation: Prozess stoppen und `state-v1.json`
   sichern. **Nur** den Programmordner
   `%LOCALAPPDATA%\Programs\AtlasMailPortable` beziehungsweise
   `~/Applications/AtlasMailPortable` entfernen, nicht das separate
   Datenverzeichnis. Dass die Datendatei unveraendert existiert, pruefen.
   Das gepruefte ZIP erneut nach obiger Anleitung installieren, starten und
   das gespeicherte Konto erwarten. Fuer einen spaeteren Versionswechsel
   ebenso stoppen, sichern und den gesamten Programmordner ersetzen;
   unbekannte Datenversionen duerfen nicht automatisch ueberschrieben werden.

Die Tests hier pruefen portable Pakete. Native Installer, Signierung,
vollstaendige Barrierefreiheit und die ausstehende Lizenz-/Plattformmatrix
bleiben entsprechend [docs/verification.md](docs/verification.md) separat offen.
Ein Entfernen der Programmdaten ist fuer die Deinstallation nicht notwendig.

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
