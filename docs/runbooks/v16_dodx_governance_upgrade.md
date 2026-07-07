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

- Commit: `f93871038f77149787230a59745257b5bca95e5f`
- Branch: `v15-1-no-vote-upgrade`
- Build host: inactive/test host, Linux amd64
- Go: `go1.24.7`
- Build command: `CGO_ENABLED=1 LEDGER_ENABLED=false make build`
- Binary SHA256: `62b47006b19a367ae1ca9420c80cc28c9512b6eadb9ff05e3fd641fa157b8b6e`

The binary has been staged on NodeNexus and Classicnodes as:

```bash
/home/dochain/go/bin/dochaind-f938710-v16
```

It has been staged on DoFoundation as:

```bash
/usr/local/bin/dochaind-f938710-v16
```

It has not replaced the live binary.

## Do Not Restart Early

Do not replace the live `dochaind` binary and restart before the planned
upgrade height. The new binary mounts a new KV store, so nodes should switch at
the `v16` upgrade height after the old binary has halted for the software
upgrade.

## Rollout

1. Build or copy the exact binary to every validator and RPC/sentry node.
2. Verify the hash on every node:

```bash
sha256sum /path/to/dochaind-f938710-v16
```

3. Submit and pass a software-upgrade proposal named exactly `v16`.
4. At the upgrade height, stop each old daemon after it halts for the upgrade.
5. Back up the existing binary:

```bash
cp -a /home/dochain/go/bin/dochaind /home/dochain/go/bin/dochaind.pre-v16
```

6. Replace the binary atomically:

```bash
install -o root -g root -m 0755 /home/dochain/go/bin/dochaind-f938710-v16 /home/dochain/go/bin/dochaind
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
`dochain` user. DoFoundation runs `dochaind` directly via systemd as `root`.
MainFCD still needs access before the v16 binary can be staged everywhere.
