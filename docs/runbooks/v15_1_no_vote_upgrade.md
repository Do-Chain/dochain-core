# v15_1 No-Vote Upgrade Runbook

This upgrade is a coordinated hard fork. Do not restart validators with the new
binary until the activation height and CosmWasm upload policy are confirmed.

## Decisions To Confirm

- `types/fork.DoCommunityGovernanceHeight`: set from `0` to the agreed future block height.
- `app/upgrades/v15_1.cosmWasmUploadAllowlist`: leave empty for `Everybody`, or fill with exact `do...` addresses for `AnyOfAddresses`.
- Restitution source: use an archive LCD or restored node that can serve heights `503999` and `508033`.

## What Activates At The Fork Height

- Governance proposal deposits become `1000000udodx`.
- Community voting power is capped at 2.5% per wallet.
- Bonded validator operators cannot vote in Community voting; they only count once in phase-one validator backing.
- CosmWasm upload and default instantiate access are updated by the selected upload policy.
- Oracle and standard slashing fractions are set to zero so jail events no longer burn delegator stake.

## Build

Build on Linux with Go `1.24.7`, `CGO_ENABLED=1`, and a working C compiler.
Windows without `gcc` cannot build the CosmWasm keeper path.

```bash
git clone https://github.com/Do-Chain/dochain-core.git
cd dochain-core
git checkout <upgrade-branch>

go test ./x/oracle/keeper ./x/oracle
go test ./custom/gov
CGO_ENABLED=1 LEDGER_ENABLED=false make build
./build/dochaind version --long
sha256sum ./build/dochaind
```

## Restitution Report

Run against an archive LCD or restored node:

```bash
python ../tools/calc_jail_restitution.py \
  --lcd http://127.0.0.1:1317 \
  --rows-csv jail_restitution_rows.csv \
  --summary-csv jail_restitution_summary.csv
```

Review the CSV before paying from treasury. The report is an apparent-loss
calculation and does not automatically exclude voluntary unbonding or
redelegation after the pre-jail height.

## Validator Rollout

1. Back up each node home, including `config`, `data/priv_validator_state.json`, and service files.
2. Install the new `dochaind` binary on all validators and sentry/RPC nodes without starting it early.
3. Stop all nodes at the agreed halt window before `DoCommunityGovernanceHeight`.
4. Replace binaries atomically and verify `dochaind version --long`.
5. Start validators first, then sentry/RPC/API nodes.
6. Watch logs through the activation height for `applying fork v15_1`.
7. Query live params after activation:

```bash
dochaind query gov params -o json
curl -s http://127.0.0.1:1317/cosmwasm/wasm/v1/codes/params
```
