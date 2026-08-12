# DODEX Single Distribution-System Migration

This is a transaction-prep checklist only. Do not move user or treasury funds as part of the admin migration. Broadcast only after the new admin and treasury wallet or multisig addresses have been reviewed and backed up.

## Live Addresses

- Factory: `do1tf2zzx0ns0chv25zfphqcjjp3m08gujkwqudaaq8euhducxlw0kqxs3pms`
- Current DEX config admin/reward wallet to remove if its seed is unavailable: `do17s94j4eu9hkw238v84zl7rql7r8peqpdg9femw`
- Current wasm admin able to migrate factory and live pools: `do1f2ywc5wxpxrh27rdtt0t4alkal7320n6z6fqcu`
- DODx staking module account: `do1jfyzpxgccjp9r2ytlxv748fnnqempalp2wewkf`

## Admin Migration

1. Create and verify a new DEX admin and treasury wallet, preferably multisigs.
2. Query the current factory, router, and pair configs before changing anything.
3. Use the wasm admin migration path if the current DEX config admin key is unavailable.
4. Keep the live and future fee split as equal thirds:
   - `lp_share = 0.333333333333333333`
   - `treasury_share = 0.333333333333333333`
   - `gov_share = 0.333333333333333334`
5. Set `treasury` to the reviewed treasury wallet and `gov_rewards` directly to the DODx staking module account: `do1jfyzpxgccjp9r2ytlxv748fnnqempalp2wewkf`.
6. This direct module-account route requires the v18 upgrade, which registers `udo` as a DODx staking reward denom and enables BeginBlock reward syncing.
7. Run every message with `--generate-only` first where possible, review the JSON, then have the wasm/admin signer approve it.
8. After broadcast, re-query all touched contracts and record the transaction hashes.

Do not include any bank `MsgSend` in this migration batch. Contract ownership/config changes are enough.

## DODx Reward Distribution

The chain module already exposes the correct accounting entry point:

```bash
dochaind tx dodxstaking deposit-rewards 123456udo --from <fee-distributor> --chain-id Do-Chain
```

That command creates `MsgDepositRewards`, which sends rewards into `x/dodxstaking` and updates the reward accumulator.

Recommended native distributor flow after v18:

1. Route only the DEX `gov_rewards` fee leg directly to the DODx staking module account.
2. The DEX reward fee should be `udo`.
3. BeginBlock automatically notices new `udo` on the module account and credits the reward accumulator.
4. Query `dochaind q dodxstaking reward-pool` and representative `pending-rewards` before and after DEX trading activity.
5. Alert if DEX reward fees have been sent but the reward-pool does not increase within a few blocks.

Public reward deposits and native reward syncing currently validate to `udo` only, so non-DO fee assets must be converted before they can become DODx staking rewards.
