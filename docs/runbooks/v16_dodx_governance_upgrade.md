# v16 DODx Governance Staking Upgrade Runbook

This is a consensus software upgrade. It adds the `dodxstaking` store and
switches community governance voting power to staked DODx only.

## What Activates

- DO staking remains normal Cosmos staking: it secures validators and receives
  distribution rewards funded by fees.
- DO staking and delegations no longer provide community governance voting
  power after `v16` activates.
- DODx can be staked with the new `dodxstaking` module.
- Staked DODx provides governance voting power.
- DODx staking does not receive rewards.

## Verified Build

- Commit: `82d794fde965c94da580edcc116e8afa325cb989`
- Branch: `v15-1-no-vote-upgrade`
- Build host: inactive/test host, Linux amd64
- Go: `go1.24.7`
- Build command: `CGO_ENABLED=1 LEDGER_ENABLED=false make build`
- Binary SHA256: `4c0104bfdffdf4cf7cdf21d23d68cee025e01fc09a0c0522e0813e9866531728`

The binary has been staged on NodeNexus and Classicnodes as:

```bash
/home/dochain/go/bin/dochaind-82d794f-v16
```

It has been staged on DoFoundation and MainFCD as:

```bash
/usr/local/bin/dochaind-82d794f-v16
```

This binary replaced the live binary on all four main nodes during the manual
activation at height `1991100`.

## Manual Activation Completed

- Target height: `1991100`
- Activation time: about `2026-07-07T16:27:26Z`
- Activation method: every main node used a root-owned watcher script that
  stopped `dochaind` at height `1991099`, wrote
  `data/manual-v16-upgrade.json`, backs up the live binary, installs the staged
  `82d794f` binary, and restarts.
- Verified applied height: `1991100`
- Verified post-upgrade height: `1991162`
- Verified post-upgrade app hash:
  `2D4F482E3AC7921C05D5397C6B7C197B76547F9A80D8A827E5FED5F18F448492`

The manual activation hook exists because normal governance voting cannot
complete in a 30 minute window. The hook uses the same v16 upgrade handler as a
normal software upgrade, but reads the agreed height from the local manual plan
file.

## Do Not Restart Early

Do not replace the live `dochaind` binary and restart before the planned
upgrade height. The new binary mounts a new KV store, so nodes should switch at
the `v16` upgrade height after the old binary has halted for the software
upgrade.

## Rollout

1. Build or copy the exact binary to every validator and RPC/sentry node.
2. Verify the hash on every node:

```bash
sha256sum /path/to/dochaind-82d794f-v16
```

3. Submit and pass a software-upgrade proposal named exactly `v16`.
4. At the upgrade height, stop each old daemon after it halts for the upgrade.
5. Back up the existing binary:

```bash
cp -a /home/dochain/go/bin/dochaind /home/dochain/go/bin/dochaind.pre-v16
```

6. Replace the binary atomically:

```bash
install -o root -g root -m 0755 /home/dochain/go/bin/dochaind-82d794f-v16 /home/dochain/go/bin/dochaind
```

7. Restart the daemon and watch logs for the `v16` upgrade handler.
8. After the chain resumes, verify:

```bash
dochaind query dodxstaking total-staked --home /home/dochain/.do
dochaind query upgrade applied v16 --home /home/dochain/.do
```

## Current Access Notes

The inactive/test host can build the binary and is not running a chain service.
NodeNexus and Classicnodes run `dochaind` directly via systemd under the
`dochain` user. DoFoundation and MainFCD run `dochaind` directly via systemd as
`root`. The v16 binary is staged on all four main nodes and is not active yet.
