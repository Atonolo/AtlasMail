# Contributing

Atlas Mail is a new, local-first application. Please discuss substantial
protocol, security, persistence or suite-interface changes before implementing
them. Keep unfinished features explicitly unavailable; do not introduce
successful-looking stubs or required cloud accounts.

Use synthetic accounts and fixtures only. Do not commit credentials, private
mail, OAuth tokens or personal data. Add Go contract and failure tests for
changes to the versioned model or adapters, and preserve the last valid
persisted state on rejected writes and migrations. Before adding a Go module,
verify exact source/artifact licenses and MPL-2.0 compatibility, update the
reviewed version/license fingerprints, and regenerate `go.sum`. Free non-OSI
or public-domain components require an individual source/rights/compatibility
review under the issue policy, not a blanket exception. Run the local gate
`go run ./cmd/verify -race -all-platforms` (race requires a C compiler).
Do not create, edit, activate or run GitHub workflows for this work.
Setup and current platform-validation limits are
in [README.md](README.md) and [docs/verification.md](docs/verification.md).

Contributions to this repository are provided under the
[Mozilla Public License 2.0](LICENSE).
