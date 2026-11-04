# Third-party and license status

Atlas Mail source is licensed under MPL-2.0. The Go 1.27.3 standard library
and toolchain are BSD-3-Clause. The product does not bundle a browser engine,
Node.js, SQLite, or an installer framework; standalone mode uses the user's
operating-system browser.

The only direct Go dependency is `github.com/zalando/go-keyring` v0.2.6
(MIT AND Apache-2.0), used for native operating-system credential stores.
The macOS implementation retains Google's Apache-2.0 source notice. The complete
locked module graph checked by `go run ./cmd/licenseaudit` is:

| Module | Version | License |
|---|---:|---|
| `github.com/zalando/go-keyring` | 0.2.6 | MIT AND Apache-2.0 |
| `al.essio.dev/pkg/shellescape` | 1.5.1 | MIT |
| `github.com/danieljoos/wincred` | 1.2.2 | MIT |
| `github.com/davecgh/go-spew` | 1.1.1 | ISC |
| `github.com/godbus/dbus/v5` | 5.1.0 | BSD-2-Clause |
| `github.com/google/shlex` | 2019-12-02 | Apache-2.0 |
| `github.com/pmezard/go-difflib` | 1.0.0 | BSD-3-Clause |
| `github.com/stretchr/objx` | 0.5.2 | MIT |
| `github.com/stretchr/testify` | 1.9.0 | MIT |
| `golang.org/x/sys` | 0.26.0 | BSD-3-Clause |
| `gopkg.in/yaml.v3` | 3.0.1 | MIT AND Apache-2.0 |

Some modules in the graph are test dependencies of keyring's transitive
packages and are included in the conservative audit. The license audit
maintains a source-controlled version and license-fingerprint allowlist,
rejects module replacements, runs `go mod verify`, fails on unreviewed modules
or missing/mismatched license texts, and produces the checked component list.
`go.sum` pins the exact downloaded module contents. Generate the build-time
report with `go run ./cmd/licenseaudit` and the CycloneDX SBOM with
`go run ./cmd/licenseaudit -sbom`.

The audit also checks the exact Go 1.27.3 toolchain version, its BSD-3-Clause
license, and both its runtime (`src/vendor/modules.txt`) and build-tool
(`src/cmd/vendor/modules.txt`) inventories. Runtime entries:

| Go toolchain module | Version | License |
|---|---:|---|
| `golang.org/x/crypto` | `v0.52.1-0.20260526024921-9beb694f9766` | BSD-3-Clause |
| `golang.org/x/net` | `v0.55.1-0.20260731170536-c1d18010be90` | BSD-3-Clause |
| `golang.org/x/sys` | `v0.45.0` | BSD-3-Clause |
| `golang.org/x/text` | `v0.37.0` | BSD-3-Clause |

Build-tool entries are `github.com/google/pprof` (Apache-2.0, including
BSD-3-Clause SVGpan), `github.com/ianlancetaylor/demangle`, `rsc.io/markdown`,
and `golang.org/x/{arch,build,mod,sync,sys,telemetry,term,text,tools}`
(BSD-3-Clause). Their exact pseudo-versions, artifact relationships and
reviewed license hashes are pinned in `cmd/licenseaudit/main.go` and emitted
in the SBOM/report. The audit includes tool-only components conservatively;
their presence does not mean they are linked into each application executable.

MIT, BSD, ISC and Apache-2.0 allow source/binary redistribution and modification
with their notice obligations. Keep those licenses and attribution with
redistribution. These permissive dependencies are compatible with the
application's MPL-2.0 file-level copyleft; MPL-covered source modifications
remain subject to MPL and must be made available to recipients. Apache patent
terms are not replaced by MIT. No relicensing of third-party source is implied.
`go run ./cmd/licenseaudit -notices` emits original LICENSE/COPYING/NOTICE
texts, Go vendor patent notices and the retained macOS source notice.
Local portable packages include this output alongside the SBOM, project
LICENSE and this document.

The former Electron runtime's Ooura-2001 original-only redistribution terms
were unacceptable; replacing the stack removes those application dependencies.
The current issue separately permits verified public-domain/free non-OSI
components, but that is not blanket approval of former SQLite, IJG or browser
artifacts. No such exception is used by this Go module allowlist.

**Review scope:** a pinned module/toolchain inventory is not proof of every
per-file license, host compiler, external browser or OS service. The browser
used for diagnostic UI testing has not received artifact-complete license
approval. These remaining reviews block a claim of full issue/release
acceptance; the portable ZIPs are development evidence, not an approved release.
No GitHub workflow is required or permitted for this local gate.
