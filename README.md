# Atlas Mail

Atlas Mail ist das lokale Mail-/Kalenderfundament der Atlas-Office-Suite.
Dieses Repository bearbeitet [Atonolo/OpenSourceIdeas#5](https://github.com/Atonolo/OpenSourceIdeas/issues/5);
das bereits vorhandene, zuvor leere Repository `Atonolo/AtlasMail` wurde dort
am 2026-11-03 ausdrücklich als Ziel ausgewählt.

> **Entwicklungsstand:** Nur synthetische Testkonten verwenden. Mailversand,
> Synchronisierung, Kalender, Kontakte und Aufgaben sind noch nicht
> implementiert. Windows wurde lokal und Linux in einer isolierten VM
> teilweise geprüft; macOS- und weitere Abnahmen fehlen. Außerdem enthält das gewählte
> Electron-Paket nicht OSI-freigegebene Bestandteile. Ihre Einbindung
> wurde anhand der offiziellen Debugsymbole für alle vier vorgesehenen
> Plattform-/Architekturziele bestätigt. Die Lizenzprüfung blockiert deshalb
> und das P0 ist nicht abgenommen.

## Deutsch

### Voraussetzungen und Entwicklung

Vorgesehen sind Node.js 26.13.0 mit npm 11.20.0 sowie Windows, macOS oder
Linux. Electron 45.2.1 und electron-builder 26.18.0 werden als
Entwicklungsabhängigkeiten installiert. Für Linux sind außerdem die
Desktop-Bibliotheken erforderlich, die Electron voraussetzt; die Smoke-Tests
benötigen eine grafische Sitzung oder `xvfb-run`. Für den DEB-Build wird
zusätzlich `ar` aus dem Paket `binutils` benötigt.

```text
npm ci --no-audit --no-fund
npm test
npm run smoke:desktop
npm run smoke:web
npm run smoke:sandbox
npm start
```

Die npm-Auflösung ist in `package-lock.json` festgehalten. Bei Windows
PowerShell mit gesperrtem `npm.ps1` stattdessen `npm.cmd` mit denselben
Argumenten verwenden. Die Smoke-Tests verwenden ausschließlich temporäre
synthetische Daten. Bei Linux ohne grafische Sitzung die drei Smoke-Befehle
jeweils mit `xvfb-run -a` starten. `npm run smoke:credentials` prüft zusätzlich
die Verschlüsselung, das Lesen und Löschen mit dem tatsächlichen
OS-Schlüsselspeicher; ohne verfügbaren Schlüsselspeicher schlägt der Test
absichtlich fehl. Unter Linux prüft
`ATLAS_MAIL_EXPECT_BASIC_TEXT=1 xvfb-run -a npm run smoke:credentials`
zusätzlich, dass der Klartext-Fallback verweigert wird.

Ubuntu 24.04 kann unprivilegierte User-Namespaces durch AppArmor einschränken.
Für die ausführbare Electron-Datei im Entwicklungsverzeichnis ist dann ein
eng begrenztes AppArmor-Profil erforderlich; die CI-Konfiguration zeigt ein
Beispiel für einen kurzlebigen Runner. Die Sandbox darf nicht mit
`--no-sandbox` umgangen werden; die Anwendung verweigert diesen Start.

Optional prüft `npm run smoke:web:firefox` den echten Webdienst in einem
isolierten, headless gestarteten Firefox. Ein vom Standardpfad abweichendes
Firefox-Programm kann über `FIREFOX_BINARY` als absoluter Pfad angegeben
werden. Erforderlich ist die native ausführbare Datei einer in sich
geschlossenen Distribution; Launcher-Skripte und externe symbolische Links
werden für den Lizenznachweis nicht akzeptiert. Der Test schreibt bei Erfolg
`reports/firefox-runtime.json`, das die Lizenzprüfung mit dem aktuellen
Browser abgleicht. Er verwendet synthetische Eingaben und ersetzt keine
Tastatur- oder Screenreader-Abnahme. Firefox ESR 153.6 wurde unter Windows und
Linux, Firefox 159 unter Windows geprüft; macOS ist noch offen.

### Lokaler Webdienst

Der Dienst ist ein früher, selbst betriebener UI-/Kern-Durchstich, **kein**
implementierter IMAP-/DAV-Proxy. Er lauscht ausschließlich auf
`127.0.0.1`, verlangt für die API ein zufälliges Zugriffstoken und sendet
keine Telemetrie. Beispielsweise unter Windows PowerShell:

```powershell
$env:ATLAS_MAIL_WEB_TOKEN = node -p "require('node:crypto').randomBytes(32).toString('base64url')"
Set-Clipboard -Value $env:ATLAS_MAIL_WEB_TOKEN
npm.cmd run server
```

Unter macOS/Linux kann das Token ebenso gesetzt werden:

```sh
export ATLAS_MAIL_WEB_TOKEN="$(node -p "require('node:crypto').randomBytes(32).toString('base64url')")"
npm run server
```

Die Seite `http://127.0.0.1:3123/` öffnen und das Token aus der lokalen
Umgebung einfügen. Es bleibt nur bis zum Trennen/Schließen im Arbeitsspeicher
des Browsers. Zwischenablage und Umgebungsvariablen sind für andere Prozesse
des Benutzerkontos möglicherweise lesbar; das Token vertraulich behandeln
und nach Bedarf rotieren. `ATLAS_MAIL_PORT` ändert den Port. Ein fehlendes
oder ungültig geformtes Token verhindert den Start.

Für Zugriff über andere Geräte muss ein selbst verwalteter HTTPS-Reverse-Proxy
vor den lokalen Dienst geschaltet werden; `ATLAS_MAIL_PUBLIC_ORIGIN` muss dann
exakt die öffentliche HTTPS-Origin sein, etwa `https://mail.example.invalid`.
Der Proxy muss TLS erzwingen, den ursprünglichen `Host` unverändert
weiterreichen, unnötige Zugriffe unterbinden und darf Authorization-Header
nicht protokollieren. Der Dienst selbst darf nicht direkt ins Netzwerk
freigegeben werden. Ein einzelnes Betreiber-Token ist keine
Mehrbenutzerverwaltung; Produktionsauthentisierung und der eigentliche
Protokolladapter gehören zu
[Atonolo/OpenSourceIdeas#35](https://github.com/Atonolo/OpenSourceIdeas/issues/35).

### Daten, Sicherung und Wiederherstellung

Standardmäßig liegt `state-v1.json` unter `%APPDATA%\AtlasMail`,
`~/Library/Application Support/AtlasMail` oder
`${XDG_DATA_HOME:-~/.local/share}/AtlasMail`. `ATLAS_MAIL_DATA_DIR` erlaubt ein
absolutes alternatives Verzeichnis. Desktop und Webdienst verwenden bei
gleicher Konfiguration denselben Speicher und verweigern konkurrierende
Schreibvorgänge.

Zum Sichern Anwendung und Webdienst beenden und die Datei `state-v1.json`
an einen geschützten Ort kopieren. Zum Wiederherstellen zunächst eine Kopie
des aktuellen Stands behalten und bei beendeter Anwendung die gültige
Sicherung zurückkopieren. Für programmgesteuerte Sicherungen stehen
`StateStore.exportBackup()` und `restoreBackup()` bereit; sie prüfen Schema
und Referenzen. Bei defekten aktuellen Daten ist die manuelle Wiederherstellung
bei beendetem Prozess erforderlich. Sicherungen können persönliche
Konfigurationsdaten enthalten und sind nicht automatisch verschlüsselt.

Credentials werden getrennt gespeichert und sind an den jeweiligen
OS-Schlüsselspeicher bzw. das Benutzerprofil gebunden. Eine Kopie der
Zustandsdatei enthält sie nicht und stellt sie auf einem anderen Gerät nicht
wieder her. Dieses P0 fragt keine Credentials ab.

Nach einem Absturz kann eine `.lock`-Datei verbleiben. Nur nachdem
**sämtliche** Atlas-Mail-Prozesse beendet wurden, darf die zugehörige
Lockdatei manuell entfernt werden. Bei `UNSUPPORTED_SCHEMA`,
`INVALID_STORED_DATA`, `WRITE_FAILED` oder `STORE_LOCKED` die Fehlermeldung
beachten; ungültige Daten werden nicht als leerer Zustand überschrieben.
Bei `COMMIT_UNCERTAIN` kann das Speichern bereits erfolgreich gewesen sein:
den Datenstand vor einem erneuten Versuch prüfen.
Die atomare Ersetzung setzt ein lokales Dateisystem mit atomarem Rename
voraus; garantierte Stromausfallsicherheit ist nicht nachgewiesen.

### Pakete und Lizenzprüfung

`npm run package -- --publish never` ist für einen Build auf dem jeweiligen
Zielsystem konfiguriert: NSIS unter Windows, DMG unter macOS sowie DEB unter
Linux. Ein AppImage wird nicht mehr erzeugt: Der untersuchte Paketierer
startete es unter Ubuntu ungefragt mit `--no-sandbox`. Die Pakete sind bis zum Nachweis aus
[docs/verification.md](docs/verification.md) keine freigegebenen Installer;
macOS-Signierung/Notarisierung und automatische Updates sind nicht eingerichtet.
Deinstallation soll den persönlichen Datenordner erhalten. Dies wurde für
einen unsignierten NSIS-Installer auf einem lokalen Windows-Host und für ein
älteres DEB in einer isolierten Linux-VM geprüft. Der korrigierte Linux-Build,
macOS und die CI-Artefakte müssen noch nachgewiesen werden.

`npm run license:check` prüft installierte npm-Paketmetadaten gegen eine
konservative Allowlist und schlägt bei unbekannten Ausdrücken fehl.
Zuvor muss `npm run smoke:web:firefox` mit demselben `FIREFOX_BINARY`
erfolgreich ausgeführt worden sein, damit der geprüfte Browserstand vorliegt.
`npm run sbom` erzeugt eine CycloneDX-npm-SBOM. Die Prüfung schlägt derzeit
wegen der in Electron/Chromium eingebundenen Ooura-FFT- und IJG-JPEG-Bestandteile,
des in Chromium und Node enthaltenen gemeinfreien SQLite ohne OSI-Lizenz
und der in den geprüften Windows- und Linux-Firefox-Testlaufzeiten enthaltenen IJG- und
SQLite-Bestandteile sowie der noch ausstehenden Binärprüfung fehl. Vor einer
Freigabe müssen diese Abhängigkeiten gemäß der OSI-Vorgabe entfernt oder
ersetzt werden. Die vollständige Lizenzprüfung über alle Zielplattformen und
externen Build-/Testwerkzeuge steht ebenfalls aus; siehe
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## English

### Setup and operation

Use Node.js 26.13.0/npm 11.20.0, then run `npm ci --no-audit --no-fund`,
`npm test`, `npm run smoke:desktop`, `npm run smoke:web`,
`npm run smoke:sandbox` and `npm start`.
On Windows PowerShell, use `npm.cmd` if `npm.ps1` is blocked. On headless
Linux prefix the UI smoke commands with `xvfb-run -a`. The additional
`npm run smoke:credentials` requires an available native credential store.
On Linux, `ATLAS_MAIL_EXPECT_BASIC_TEXT=1 xvfb-run -a npm run smoke:credentials`
checks that the plaintext fallback is rejected. Building the DEB requires
`ar` from `binutils`. Ubuntu's AppArmor restrictions may require an
executable-specific user-namespace profile for development; the CI workflow
shows one for an ephemeral runner. Do not bypass the sandbox with
`--no-sandbox`: the application refuses that launch.
The optional `npm run smoke:web:firefox` uses an isolated headless Firefox;
set `FIREFOX_BINARY` to its absolute executable path when necessary. It uses
a native executable from a self-contained distribution, rejects external
symlinks, and writes `reports/firefox-runtime.json` for the license check.
It uses synthetic field input and does not replace keyboard or screen-reader
testing. Firefox ESR 153.6 was checked on Windows and Linux, and Firefox 159
on Windows; the inspected runtimes contain IJG JPEG and public-domain SQLite.
The lockfile pins npm
dependencies, but Electron also contains non-OSI-approved components
confirmed by its official debug symbols. The release license check requires
a prior successful Firefox smoke with the same `FIREFOX_BINARY` and blocks.
Linux packaging targets DEB only: the examined AppImage builder silently
added `--no-sandbox` on Ubuntu.

For the local web foundation, set `ATLAS_MAIL_WEB_TOKEN` to a random 32-byte
base64url value, run `npm run server` and visit
`http://127.0.0.1:3123/`. The service binds to loopback only; an HTTPS reverse
proxy and an exact `ATLAS_MAIL_PUBLIC_ORIGIN` are required for remote access.
The single operator token is not multi-user authentication. No real mail or
DAV protocol adapter is implemented.

To back up, stop the application and service and copy `state-v1.json` from
the platform data directory described above. Keep the existing file before
restoring a valid backup. Backups exclude credentials and may contain
personal configuration. Do not remove a `.lock` file until every Atlas Mail
process has stopped. See [the architecture decision](docs/adr/0001-foundation.md),
[contracts](docs/contracts-v1.md), [security model](docs/security.md), and
[verification status](docs/verification.md).

## License

This Source Code Form is subject to the Mozilla Public License, v. 2.0; see
[LICENSE](LICENSE). Contributions are described in [CONTRIBUTING.md](CONTRIBUTING.md).
