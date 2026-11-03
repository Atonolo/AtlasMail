# Contributing

Atlas Mail is a new, local-first application. Please discuss substantial
protocol, security, persistence or suite-interface changes before implementing
them. Keep unfinished features explicitly unavailable; do not introduce
successful-looking stubs or required cloud accounts.

Use synthetic accounts and fixtures only. Do not commit credentials, private
mail, OAuth tokens or personal data. Add Go contract and failure tests for
changes to the versioned model or adapters, and preserve the last valid
persisted state on rejected writes and migrations. Before adding a Go module,
verify its OSI-approved license, update the reviewed license audit, and
regenerate `go.sum`. Run `go test ./...`, `go vet ./...` and
`go run ./cmd/licenseaudit`. Setup and current platform-validation limits are
in [README.md](README.md) and [docs/verification.md](docs/verification.md).

Contributions to this repository are provided under the
[Mozilla Public License 2.0](LICENSE).
