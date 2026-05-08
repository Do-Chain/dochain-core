# DoChain Mainnet Readiness Notes

This document records the launch-hardening assumptions expected by the external audit package.

## Genesis Policy

- Treat `dochain-1` as a public mainnet candidate, not a convenience testnet.
- Generate or harden genesis with `scripts/harden_mainnet_genesis.py`.
- Launch with at least four independent genesis validators.
- Regenerate `genesis_time` for the final launch artifact.
- Document foundation-controlled accounts, custody model, and vesting/lockup expectations before submission.

## Gated Launch Surface

- Wasm uploads are disabled at genesis: `code_upload_access.permission = "Nobody"`.
- Wasm default instantiation is disabled at genesis: `instantiate_default_permission = "Nobody"`.
- IBC clients are restricted to `07-tendermint`.
- IBC transfer send/receive are disabled until channels and relayer policy are audited.
- ICA controller and host are disabled at genesis; ICA host `allow_messages` is empty.

## Oracle And Market

- Oracle whitelist at launch is `udo` and `uusd`.
- `udo` Tobin tax is `0`.
- `uusd` Tobin tax is `0.002500000000000000`.
- Genesis exchange rates remain empty; validators must run feeders before market swaps involving `uusd` are relied on.
- Price-server sources and feeder deployment variables must be submitted with the final audit bundle.

## Build Environment

- Final audit build should be Linux with Go `1.24.7`, CGO enabled, and WasmVM build dependencies installed.
- Windows without `gcc` cannot validate Wasmd-dependent packages because the no-CGO keeper path is not compatible with Wasmd helper code.
