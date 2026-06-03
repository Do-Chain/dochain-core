# DoChain Deployment Plan: 3 Nodes Plus 1 Endpoint

This plan is for the current four-server constraint: three validator nodes and one public endpoint/full node.

## Important Topology Warning

Three equal-power validators are not a resilient mainnet topology. CometBFT needs more than two thirds of voting power to make progress. With three equal validators, one validator going offline leaves exactly two thirds online, which is not enough to finalize blocks.

Use this topology for staging or a controlled beta. For a resilient public mainnet, use at least four independent validators, ideally with an endpoint/sentry layer that is separate from validator signing machines.

## Roles

- `validator-1`, `validator-2`, `validator-3`: hold validator keys, sign blocks, expose P2P only.
- `endpoint-1`: no validator signing key; exposes public RPC/API/gRPC with rate limiting and monitoring.
- Oracle feeder: run per validator, preferably on the validator host or a private operations host with LCD access.
- Price server: keep private to feeders unless there is a deliberate public data API reason.

## Network Exposure

Validators:

- Public inbound: `26656/tcp` P2P only.
- Private/admin inbound: `22/tcp` from your admin IP only.
- Bind RPC/API/gRPC to localhost: `26657`, `1317`, `9090`.
- Do not expose validator RPC/API/gRPC to the internet.

Endpoint:

- Public inbound: `26657/tcp`, `1317/tcp`, `9090/tcp` if the wallet and public clients need them.
- Add reverse proxy/rate limits in front of REST/RPC where possible.
- No validator private key on this machine.
- It can expose `26656/tcp` for peer discovery if you want it to act as a sentry/full node.

Oracle:

- Do not expose feeder keys, `voter.json`, `ORACLE_FEEDER_PASSWORD`, or mnemonics.
- Feeders should read from private or endpoint LCD URLs and private price-server URLs.
- Price-server upstream calls have a default timeout; tune with `PRICE_SERVER_FETCH_TIMEOUT_MS` if needed.

## Build Gate

Build the deployment binary on Linux with CGO enabled. The Windows no-CGO path is not valid for the current Wasmd v3 keeper signature.

```bash
make build-linux
./build/dochaind version
```

If Docker is unavailable, build on a Linux host with Go `1.24.7`, CGO enabled, gcc/build-essential, and WasmVM v3 static library support.

Do not use `shared.Dockerfile`; it is intentionally deprecated.

## Genesis Gate

Generate genesis from the selected validators, then harden it:

```bash
python3 scripts/harden_mainnet_genesis.py genesis.json genesis.hardened.json --genesis-time now --strict
```

`--strict` should fail with only three validator gentxs. That failure is correct for mainnet readiness. For a controlled beta with three validators, run without `--strict`, record the warning, and accept that one validator outage can halt the chain.

Required hardened defaults:

- Wasm upload disabled.
- Wasm default instantiate disabled.
- IBC transfer send/receive disabled.
- ICA host/controller disabled.
- IBC clients restricted to `07-tendermint`.
- Oracle whitelist limited to `udo` and `uusd` unless separately audited.

## Launch Steps

1. Provision all four servers with a non-root `dochain` user, locked-down SSH, host firewall, NTP/time sync, and log rotation.
2. Build the Linux binary or image from the audited branch.
3. Create validator keys on validator hosts only; keep `priv_validator_key.json` off the endpoint.
4. Collect gentxs, assemble genesis, run the hardening script, and confirm the same genesis SHA-256 on every node.
5. Configure validator P2P `persistent_peers` to include the other validators and the endpoint.
6. Configure endpoint P2P to peer with validators, then expose only the public client ports required.
7. Start validators first, then endpoint.
8. Start price-server and validator oracle feeders after the first blocks are live.
9. Verify block production, oracle votes, endpoint queries, and MFA-protected transaction rejection/approval paths.

## Pre-Launch Acceptance Checks

- `go test` passes for the targeted consensus/security paths documented in the audit notes.
- SDK gov/staking/bank and touched keeper packages pass under Go `1.24.7`.
- Oracle root Jest suite passes, feeder/price-server TypeScript compile, and production `npm audit` is clean.
- `govulncheck` is clean for the SDK targeted paths.
- Core `govulncheck` still reports `GO-2026-4740` through WasmVM/msgpack; mitigate by keeping public Wasm upload and instantiate disabled until upstream fixes are available.
- No public validator RPC/API/gRPC exposure.
- Endpoint has rate limits, monitoring, and backups.
- Genesis hash is identical on all nodes.
