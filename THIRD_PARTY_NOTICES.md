# Third-party and license status

Atlas Mail source is licensed under MPL-2.0. The Go 1.27.3 standard library
and toolchain are BSD-3-Clause. The product does not bundle a browser engine,
Node.js, SQLite, or an installer framework; standalone mode uses the user's
operating-system browser.

The only direct Go dependency is `github.com/zalando/go-keyring` v0.2.6
(MIT), used for native operating-system credential stores. The complete
locked module graph checked by `go run ./cmd/licenseaudit` is:

| Module | Version | License |
|---|---:|---|
| `github.com/zalando/go-keyring` | 0.2.6 | MIT |
| `al.essio.dev/pkg/shellescape` | 1.5.1 | MIT |
| `github.com/danieljoos/wincred` | 1.2.2 | MIT |
| `github.com/davecgh/go-spew` | 1.1.1 | ISC |
| `github.com/godbus/dbus/v5` | 5.1.0 | BSD-2-Clause |
| `github.com/google/shlex` | 2019-12-02 | Apache-2.0 |
| `github.com/pmezard/go-difflib` | 1.0.0 | BSD-3-Clause |
| `github.com/stretchr/objx` | 0.5.2 | MIT |
| `github.com/stretchr/testify` | 1.9.0 | MIT |
| `golang.org/x/sys` | 0.26.0 | BSD-3-Clause |
| `gopkg.in/yaml.v3` | 3.0.1 | MIT OR Apache-2.0 |

Some modules in the graph are test dependencies of keyring's transitive
packages and are included in the conservative audit. The license audit
maintains a source-controlled allowlist, fails on unreviewed modules or
missing/mismatched license texts, and produces the checked component list.
`go.sum` pins the exact downloaded module contents. Generate the build-time
report with `go run ./cmd/licenseaudit` and the CycloneDX SBOM with
`go run ./cmd/licenseaudit -sbom`.

The audit also checks the exact Go 1.27.3 toolchain version, its BSD-3-Clause
license, and every module listed in the toolchain's vendored module inventory:

| Go toolchain module | Version | License |
|---|---:|---|
| `golang.org/x/crypto` | `v0.52.1-0.20260526024921-9beb694f9766` | BSD-3-Clause |
| `golang.org/x/net` | `v0.55.1-0.20260731170536-c1d18010be90` | BSD-3-Clause |
| `golang.org/x/sys` | `v0.45.0` | BSD-3-Clause |
| `golang.org/x/text` | `v0.37.0` | BSD-3-Clause |

The former Electron runtime's non-OSI Ooura FFT, IJG JPEG and public-domain
SQLite components were the reason for replacing that architecture. They are
not dependencies of this implementation. The existing CI workflow remains
on the old stack because workflow changes were explicitly excluded; therefore
the new audit is locally runnable but is not yet a CI gate.
