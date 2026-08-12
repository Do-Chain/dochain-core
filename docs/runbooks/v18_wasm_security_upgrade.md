# v18 Wasm Security And DODx Rewards Upgrade Runbook

This is a consensus software upgrade for the live `Do-Chain` network. It must
not be activated until the same verified Linux binary and manual plan are
staged on every validator.

## Scope

At the `v18` activation height, the upgrade changes these Wasm defaults:

- `code_upload_access.permission`: `Everybody` to `Nobody`
- `instantiate_default_permission`: `Everybody` to `Nobody`

It also enables the native DEX-to-DODx-stakers reward path:

- enables `x/dodxstaking` reward accounting
- registers `udo` as the reward denom for DODx stakers
- syncs any existing module-account `udo` balance once at activation
- after activation, BeginBlock keeps syncing new `udo` sent to the DODx staking
  module account

Existing uploaded code and instantiated contracts remain available. The
upgrade does not delete code, migrate stores, or modify contract state.

The upgrade does not change:

- DODX staking or its one-to-one governance voting power
- DO validator staking, gas, fees, or distribution rewards
- governance quorum, veto, deposits, or voting periods
- oracle or Cosmos slashing parameters
- any KV-store layout

The historical `v15_1` fork must remain byte-for-byte behavior-compatible with
mainnet replay. Its historical parameter transition is not used to secure the
already-running chain; that remediation belongs to `v18`.

## Release Gate

Record these values before staging:

- commit: `<COMMIT>`
- release tag: `<TAG>`
- Linux amd64 SHA256: `<SHA256>`
- Go version: `go1.25.12`
- build command: `CGO_ENABLED=1 LEDGER_ENABLED=false make build`

Required green checks:

- Linux CGO/WasmVM build
- full `go test ./...`
- race-enabled test workflow
- upgrade/localnet test workflow
- dependency security regression tests

Do not build independently on validator hosts. Every node must receive the same
artifact and match the recorded SHA256.

## Pre-Activation Checks

1. Confirm every node reports network `Do-Chain` and is fully caught up.
2. Confirm no other upgrade plan is scheduled.
3. Export and retain the live governance, Wasm, staking, slashing, oracle, and
   DODX-staking parameters.
4. Take a recoverable data snapshot on every validator at a common finalized
   height.
5. Back up the current `v1.0.1-rc6` binary on every node.
6. Stage the new binary without replacing or restarting the live service.
7. Verify its SHA256 on every node.
8. Choose an activation height with enough time for every operator to confirm
   the artifact, snapshot, plan, monitoring, and rollback readiness.

The manual plan file is `<node-home>/data/manual-v18-upgrade.json`:

```json
{
  "name": "v18",
  "height": <HEIGHT>,
  "info": "Do-Chain v18 Wasm permission hardening and native DODx staking rewards"
}
```

The name must be exactly `v18`, and every validator must use the same height.

## Governance Proposal Template

This proposal can be submitted from any machine with:

- `dochaind`
- the proposer key loaded
- enough DODx for the proposal deposit
- enough DO for gas
- RPC access to a synced node

The signer address does not need to be on the chain server. For the wallet
`do1gw39atpxnn6n2un7msagqgj5p7yg7xumgyzpzh`, use the local key name that
resolves to that address as `FROM`.

Set `UPGRADE_HEIGHT` far enough in the future for every validator to stage the
same binary and manual plan.

```bash
export CHAIN_ID=Do-Chain
export NODE=http://178.63.79.250:26657
export FROM=<key-name-for-do1gw39atpxnn6n2un7msagqgj5p7yg7xumgyzpzh>
export KEYRING_BACKEND=test
export UPGRADE_HEIGHT=<chosen-height>

GOV_AUTHORITY="$(dochaind q auth module-account gov \
  --node "$NODE" \
  --output json \
  | jq -r '.account.value.address // .account.base_account.address // .account.address')"

cat > v18-upgrade-proposal.json <<EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "$GOV_AUTHORITY",
      "plan": {
        "name": "v18",
        "height": "$UPGRADE_HEIGHT",
        "info": "Do-Chain v18 Wasm permission hardening and native DODx staking rewards"
      }
    }
  ],
  "deposit": "20000000udodx",
  "title": "v18 Wasm security and native DODx rewards",
  "summary": "Closes permissionless Wasm upload/default instantiation and enables automatic udo reward accounting for DODx stakers.",
  "metadata": "",
  "expedited": false
}
EOF

dochaind tx gov submit-proposal v18-upgrade-proposal.json \
  --from "$FROM" \
  --keyring-backend "$KEYRING_BACKEND" \
  --chain-id "$CHAIN_ID" \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 100000udo \
  -y
```

## Activation

1. Stop each validator only at the coordinated switch point.
2. Verify the last committed height and preserve the service logs.
3. Install the verified binary atomically over the service binary.
4. Write the identical manual plan to each node's `data` directory.
5. Restart services and monitor consensus on all nodes.
6. Confirm the upgrade applied at the agreed height:

```bash
dochaind query upgrade applied v18 --home <node-home>
```

7. Confirm both Wasm permissions are `Nobody`:

```bash
dochaind query wasm params --node tcp://127.0.0.1:26657 --output json
```

8. Confirm DODX stake totals, a sample DODX governance-power query, DODx staking
   reward-pool behavior, validator status, oracle voting, RPC, LCD, and block
   production.
9. Compare all non-Wasm parameter exports with the pre-activation copies.

## DEX Reward Switch After v18

After `v18` is applied, point the DEX reward-fee leg at the DODx staking module
account:

```bash
export DODX_STAKING_REWARDS_MODULE=do1jfyzpxgccjp9r2ytlxv748fnnqempalp2wewkf
```

When applying the DEX split/migration, use:

```bash
REWARD_DISTRIBUTOR=do1jfyzpxgccjp9r2ytlxv748fnnqempalp2wewkf
```

No reward hot wallet and no `execute {"deposit_rewards":{}}` keeper are needed
for this native route.

## Rollback

Before the activation height, rollback is safe: stop the service, remove the
unapplied `manual-v18-upgrade.json`, restore the previous binary, and restart.

After `v18` has committed, do not restore the old binary on one validator or
attempt an independent state rollback. The chain has committed a consensus
state transition. Recovery must be coordinated across the validator set using
either a forward-fix binary or the same pre-upgrade snapshot on every validator.

## Exit Condition

The rollout is complete only when:

- `v18` is recorded as applied at the agreed height
- the chain produces finalized blocks without validator divergence
- both Wasm permissions are `Nobody`
- DODX governance behaves as before
- DODx staking rewards account incoming `udo` sent to the module account
- all other exported parameters match their pre-upgrade values
- RPC, LCD, oracle feeders, redemption monitoring, and DEX health checks pass
