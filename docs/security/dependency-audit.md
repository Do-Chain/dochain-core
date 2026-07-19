# Go dependency security audit

Audit date: 2026-07-19

The `dochaind` dependency graph was upgraded and re-scanned with Go 1.25.12
and the current `govulncheck` database. The following vulnerable versions were
replaced:

| Module | Selected version |
| --- | --- |
| `github.com/hashicorp/go-getter` | `v1.8.6` |
| `github.com/go-jose/go-jose/v4` | `v4.1.4` |
| `github.com/ulikunitz/xz` | `v0.5.15` |
| `golang.org/x/net` | `v0.55.0` |
| `golang.org/x/crypto` | `v0.52.0` |
| `github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream` | `v1.7.8` |
| `github.com/aws/aws-sdk-go-v2/service/s3` | `v1.97.3` |
| `github.com/CosmWasm/wasmvm/v3` | `v3.0.7` |
| `github.com/shamaton/msgpack/v2` | `v2.4.1` |

## Residual scanner findings

These findings are deliberately documented rather than suppressed.

### GO-2026-5932: `golang.org/x/crypto/openpgp`

There is no fixed `x/crypto` version for this advisory. The only imports in the
selected Cosmos SDK and CometBFT code are `openpgp/armor`. They base64-wrap and
unwrap local key files; they do not invoke OpenPGP signing, encryption, key
validation, or packet parsing. `govulncheck` confirms that the binary's symbol
traces end in ASCII-armour `Encode`, `Decode`, `Read`, and `Write` operations.
No transaction, consensus message, oracle value, or Wasm message reaches this
local CLI/keyring path.

Replacement with the maintained ProtonMail fork should be made upstream in the
Cosmos SDK and CometBFT dependencies. Maintaining a consensus-critical fork
only to change this local ASCII-armour helper would create greater upgrade and
compatibility risk. Re-audit this exception whenever either upstream changes
its keyring implementation.

Reference: <https://pkg.go.dev/vuln/GO-2026-5932>

### GO-2026-4513 / GO-2026-4740: `github.com/shamaton/msgpack/v2`

The vulnerability database currently labels all versions affected and lists no
fixed version. The selected latest release, `v2.4.1`, bounds every fixext read
through `readSizeN` in `internal/decoding/ext.go`, returning an error for a
truncated buffer. `app/dependency_security_test.go` exercises truncated
`0xd4`-`0xd8` inputs and asserts that decoding returns an error without a panic.

Keep the regression test until the advisory database publishes an explicit
fixed version, then upgrade and remove this exception.

References: <https://pkg.go.dev/vuln/GO-2026-4513> and
<https://pkg.go.dev/vuln/GO-2026-4740>

### GO-2025-3442: CometBFT module-only report

The selected CometBFT version is `v0.38.21`. The affected `blocksync` package
was fixed in `v0.38.17`; the Go vulnerability page lists that fixed version for
the `v0.38` package and `v1.0.1` for the later `internal/blocksync` package.
`govulncheck -show verbose` reports no called vulnerable symbol in this binary,
but its module summary incorrectly associates the selected `v0.38.21` module
with the `v1.0.1` threshold. A major CometBFT upgrade is therefore neither
required for this advisory nor safe to perform as a dependency-only change.

Reference: <https://pkg.go.dev/vuln/GO-2025-3442>

## Reproduction

Run from the `dochain-core` directory:

```text
go test ./...
go build ./cmd/dochaind
govulncheck ./cmd/dochaind
govulncheck -show verbose ./cmd/dochaind
```

`govulncheck` intentionally exits non-zero while the two no-fixed-version
symbol findings above remain in the database. Treat any additional finding as
a release blocker.
