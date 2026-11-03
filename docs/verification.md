# Verification and acceptance evidence

The Go redevelopment removes the Electron/Chromium and Node runtime packages
that failed the OSI-only policy. The replacement dependency graph is locked in
`go.mod`/`go.sum`, and `go run ./cmd/licenseaudit` validates its reviewed
licenses and source license text. `go run ./cmd/licenseaudit -sbom` emits a
CycloneDX SBOM.

## Reproducible local checks

| Check | Command | Status |
|---|---|---|
| Model, store, credential boundary and HTTP contract tests | `go test -race ./...` | Passed on Linux with Go 1.27.3 |
| Static analysis | `go vet ./...` | Passed on Linux with Go 1.27.3 |
| Linux x64 executable build | `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ...` | Passed |
| Windows x64 executable cross-build | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ...` | Passed |
| Windows arm64 executable cross-build | `GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build ...` | Passed |
| macOS Intel/Apple Silicon cross-builds | `GOOS=darwin GOARCH=amd64/arm64 CGO_ENABLED=0 go build ...` | Both passed |
| Linux arm64 executable cross-build | `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ...` | Passed |
| Dependency/license audit | `go run ./cmd/licenseaudit` | Passed; reviewed modules and pinned Go toolchain are OSI-licensed |
| CycloneDX SBOM | `go run ./cmd/licenseaudit -sbom` | Generated and parsed successfully; includes the Go toolchain and vendored modules |
| Real desktop-browser and native keyring operation | Run on each target OS | Not established by cross-build |
| Installer install/uninstall and data retention | Target-specific package and clean host | Not established |
| Browser matrix and WCAG 2.1 AA/screen-reader review | Windows, macOS and Linux | Not established |
| CI execution | Existing workflow dispatch | Not applicable: workflow still targets retired npm/Electron chain and was left untouched by request |

No Windows or macOS run, signing, installer, browser/OS matrix or complete
assistive-technology assessment is claimed from the Linux development host.
The existing GitHub Actions workflow must be deliberately migrated before it
can validate this foundation; it was not edited or replaced.
