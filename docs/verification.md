# Verification and release evidence

Status: **P0 acceptance is pending.** Windows has local execution and installer
evidence and Linux has scoped VM evidence, but the full target matrix has not
passed and non-OSI components confirmed in Electron's binaries block the
license gate. Do not tick the source issue's acceptance criteria based only
on this document.

On 2026-11-03, a Chromium 156 browser harness on Windows loaded the local
source files and exercised the model, adapter failures, a simulated filesystem
and credential provider, an HTTP-boundary mock, and the UI with synthetic
responses. It also checked visible text contrast heuristically and compared
the MPL license text with the official text. These are preliminary mocked
checks: the harness removed the page CSP for injection and did not establish
OS, native-keyring, real-filesystem, accessibility or package compatibility.
The separate real Windows checks below supersede it only for their stated scope.

On 2026-11-03, Windows 11 Pro x64 build 26200 with Node 24.16.0,
npm 11.13.0 and Electron 45.2.1 successfully ran a clean `npm ci`, the Node
tests, desktop and web smoke scripts, and the native DPAPI credential smoke.
The real web service was restarted and Chromium 156 reloaded a saved
synthetic account. An unsigned Windows NSIS package was built; its unpacked
application was started twice with isolated profile/data directories, and a
synthetic account survived the restart.

On the same Windows host, the exact unsigned installer
`Atlas Mail Setup 0.1.0.exe` (SHA-256
`b511d110c776d5c5755463543c7e9a53490c4ded7e69e392122b180cb75c0da9`)
was installed silently for the current user into an isolated temporary
directory. The installed application saved a synthetic account, reloaded it
after a restart, and was then uninstalled. Its default
`%APPDATA%\AtlasMail\state-v1.json` survived uninstallation in the isolated
profile. Relevant installer registry entries and shortcuts were checked before
the test and the test registrations, shortcut and temporary data were
removed afterward. This was a local Windows host, **not** a clean VM, a signed
release or evidence for another OS.

A later local rebuild of the unsigned Windows installer had SHA-256
`828654b77a9bd8fa6c9a8c48e045a3e03c9e33f1d269ae8363c76ac892ff13d5`.
It was installed per-user into a separate temporary directory. Windows UI
Automation was used to save a synthetic account in an isolated default
`%APPDATA%\AtlasMail` directory and confirm it was visible after restarting
the installed app. Uninstall removed the app, its registration and shortcuts,
and retained that state file. This is still a local-host check, not a clean-VM
or screen-reader test.

On the same host, headless Chrome 156.0.8078.80 and Edge 156.0.4314.58
used the real local service through its unchanged CSP. A Playwright harness
entered and submitted synthetic accounts by keyboard, checked the focus
indicator on the traversed controls, reconnected after reload, and observed
only the local service origin in page requests. The harness is not a packet
capture or a complete accessibility assessment.

The optional `npm run smoke:web:firefox` also passed on Windows with Firefox
159.0.1 and with an isolated, uninstalled Firefox ESR 153.6.0 (build ID
`20261022211647`). It used WebDriver BiDi pointer events and programmatically
set form values to save and reload synthetic data; it did **not** establish
keyboard or screen-reader accessibility. The ESR executable SHA-256 was
`172b7bac283c3c468f4f8a684ca24dc702e6c5552ecb4c85bfa466836bf36ff8`.
The [official Windows ESR installer](https://download.mozilla.org/?product=firefox-esr-latest-ssl&os=win64&lang=en-US)
used for isolated extraction had a valid Mozilla signature and SHA-256
`f367f2e4714ad86459e1f383848ea1633216a424aee28b4977ab0d52b4901151`.
The optional test accepts an absolute native browser executable via
`FIREFOX_BINARY` and records its exact runtime identity before and after a
successful smoke. Matching symbols confirm that both Windows Firefox
distributions contain IJG-derived JPEG and public-domain SQLite code, which
also block the issue's OSI-only test-dependency requirement. Evidence and
limitations are in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).

On 2026-11-03 an isolated Ubuntu 24.04.5 x86-64 QEMU VM with software
emulation ran the core tests and the real desktop and web smoke tests with
Node 26.13.0, npm 11.20.0 and Electron 45.2.1. The desktop and web tests
used Xvfb. Ubuntu's AppArmor user-namespace restriction remained enabled; a
test-specific profile allowed the exact development Electron executable to
create its sandbox. The corresponding CI profile setup was also executed in
the VM, but the GitHub Actions job itself has not run.

An isolated D-Bus session with an unlocked GNOME keyring passed the credential
encrypt/read/delete smoke with the actual `gnome_libsecret` backend. A
separate run forced Electron's `basic_text` backend and verified that the
credential store rejected it without writing a secret. The Linux CI step was
executed in the VM for both cases; it has not yet run on GitHub Actions.

The official Firefox ESR 153.6.0 archive, also configured for Linux CI, was
downloaded and verified against SHA-256
`31321cd9e2cef73df678417e810cdd1b5760aa5cc8063c8465669e5869cb8498`.
Its headless Linux smoke passed authentication, synthetic persistence and
reload. Matched Linux symbols confirm IJG-derived JPEG and public-domain
SQLite in that exact distribution; the license gate rejects both. This is
not a complete Linux browser or dependency audit.

Orca 46.1 on that VM observed the Electron accessibility tree through AT-SPI.
The tree exposed labelled language toggles, name/address fields, protocol
selection and the save button. During keyboard traversal, Orca's debug output
included the overview link, a language toggle and the save button. Speech
Dispatcher used a silent dummy output, so no audible output was assessed.
Not every control, error announcement, language or WCAG criterion was tested.

The initial Linux AppImage built by electron-builder 26.18.0 started with
`--no-sandbox` on the restricted Ubuntu VM; inspection of the live process
command line confirmed it. AppImage packaging was removed rather than silently
weakening isolation. The desktop now rejects `--no-sandbox` before creating
its window, and the negative smoke passed on Windows and Linux.

An earlier unsigned DEB, `atlas-mail_0.1.0_amd64.deb` with SHA-256
`38b493017dab6a6017181361e6c7756d25d2cf19921e1c857f081230aa3791d0`,
was installed in the Ubuntu VM with `apt-get install ./<artifact>`. The test
launched `/usr/bin/atlas-mail` without adding a `--no-sandbox` argument; its
window reported `WM_CLASS` `atlas-mail`, and a synthetic account entered by
keyboard was displayed again after restart. This does not by itself prove the
kernel sandbox was active. `apt-get remove atlas-mail` removed the
application and desktop entry while the isolated default
`$XDG_DATA_HOME/AtlasMail/state-v1.json` remained byte-for-byte unchanged.
That artifact predates the disabled-sandbox guard; the corrected artifact
still needs its own installer verification. This was an emulated,
test-configured VM, not a clean physical desktop or a CI artifact.

A later unsigned DEB with the disabled-sandbox guard was built with SHA-256
`335994fe0a5a6dc59456c921631f75514ad514c02e9acb6c599a1f459f5ba6c7`
and installed successfully in that VM. A resumed test with a fresh isolated
profile observed the renderer in seccomp filter mode, saved a synthetic
account and displayed it again after restarting. Uninstallation and data
retention have **not** been established for this artifact. A separate
`--no-sandbox` launch exited with status 1 and reported the error to stderr.
Electron documents that `showErrorBox` called before readiness on Linux does
not show a GUI dialog. The source now waits for Electron readiness before
showing the sandbox rejection; the regression smoke checks that ordering, but
a visible dialog has not been confirmed in the VM. The installed package still
predates this source change and the latest local documentation updates. The VM
was shut down with this package still installed.

After that source change, the complete Node/Python suite and Linux desktop,
web and disabled-sandbox smokes passed on the current Linux host with Node
26.12.1 and Electron 45.2.1. Desktop and web smokes ran as an unprivileged
user under Xvfb; the initial root-run attempts correctly failed because
Chromium refuses to start as root without disabling its sandbox. A separate
unprivileged `--no-sandbox` launch under Xvfb produced an open X11 error
dialog, confirming that the guard now reaches a GUI dialog after readiness.
The dialog text was not independently inspected, and this host-level check
does not replace testing the rebuilt DEB in the Ubuntu VM, including
uninstallation and byte-for-byte data retention.

The Windows Electron `LICENSES.chromium.html` SHA-256 was
`7dac8267ac6da8d959cfe88f5f055b18a397f10230380b5329125e167bf5639b`;
its known non-OSI FFT, IJG JPEG and public-domain SQLite notices remain
release blockers under the issue's OSI-only requirement.
Official debug symbols matched to the Windows, Linux and both macOS Electron
binaries also confirm functions from all three components are linked. The artifact
identities and limitations are in
[THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md); inspecting binaries is not
an application execution or installation test on those systems.

| Check | Reproducible procedure | Evidence/status |
|---|---|---|
| Core, adapter and failure contracts | `npm test` | Passed locally on Windows/Node 24 and in the Ubuntu VM/Node 26; macOS and CI runs pending |
| Desktop UI → core → persistence | `npm run smoke:desktop` | Passed locally on Windows and in the Ubuntu VM/Electron 45, including keyboard entry and submission; macOS pending |
| Desktop network boundary | `npm run smoke:desktop` starts a loopback listener and verifies the Electron session cannot reach it | Passed locally on Windows; full process packet-level audit pending |
| Browser → authenticated service → persistence | `npm run smoke:web`, including keyboard login/account entry and an in-flight response after disconnect | Passed locally on Windows and in the Ubuntu VM/Electron's Chromium 156; real Windows Chromium service-restart check also passed |
| Package builds | `npm run package -- --publish never` per OS | Unsigned Windows NSIS and Linux DEB built; macOS build pending |
| Windows NSIS install, launch, uninstall, data retention | In an isolated Windows profile, install the exact artifact per-user to an empty test directory, add a synthetic account, restart, uninstall and inspect `%APPDATA%\AtlasMail` | Passed on the local Windows 11 host with the artifact and limitations above; clean-VM and CI-artifact confirmation pending |
| macOS DMG install, launch, uninstall, data retention | Mount the exact CI artifact, copy the app, add a synthetic account, restart, remove the app, inspect data on Intel and Apple Silicon | Pending; record artifact hash, OS versions, signatures and outcomes |
| Linux DEB install/start/removal, data retention | Use the exact CI artifact on a clean Ubuntu desktop; add a synthetic account, restart, remove package, inspect data | Earlier DEB passed in the isolated VM as described above; later guarded DEB passed install, start and restart, but removal and retention remain pending. CI-artifact confirmation is also pending. AppImage is not a supported target |
| Browser matrix | Open the authenticated web UI on Windows/macOS/Linux in Chrome/Edge, Firefox ESR and Safari where available; add/reload a synthetic account | Windows Chrome 156, Edge 156 and Firefox ESR, plus Ubuntu Firefox ESR, passed the scoped checks above; CI, macOS and Safari pending |
| Keyboard baseline | Both Electron smoke tests tab from the skip link through navigation and language controls, check visible focus, and enter and submit synthetic data | Passed locally in Windows desktop, Electron Chromium, headless Chrome/Edge and Ubuntu Electron; Firefox and macOS pending |
| Screen reader, WCAG 2.1 AA | Navigate and operate every active control, observe focus, labels, live errors and contrast with platform assistive technology | Partial Ubuntu Orca/AT-SPI observation above; full screen-reader and WCAG assessment pending |
| Native credential protection | `npm run smoke:credentials`; test unavailable keyring and Linux plaintext fallback | Windows DPAPI and Ubuntu `gnome_libsecret` encrypt/read/delete passed; Linux `basic_text` was rejected. macOS and CI runs pending |
| Complete licenses/SBOM | Review npm and bundled binary licenses and produce release notices for each target | npm lockfile and npm SBOM generation exist; the gate rejects Ooura FFT, IJG JPEG and SQLite in Electron, SQLite in Node, IJG JPEG and SQLite in inspected Windows and Linux Firefox test runtimes, and the missing binary audit |

Record actual CI run links, versioned artifacts, hashes and manual results
here or in a linked release record when they exist. Never replace a failed
check with an unverified success or expose real account data in test evidence.
