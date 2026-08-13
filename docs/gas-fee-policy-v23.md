# DoChain Gas Fee Policy v23

v23 keeps the existing v22 direct-send value fee policy and adds successful normal gas refunds.

The valuefee params schema is intentionally unchanged from v22. v23 only changes the existing `PolicyVersion` field from `22` to `23` at the upgrade height. This keeps pre-upgrade replay compatible with v22.

Refund behavior:

- Direct wallet sends that are charged the value fee are not refunded.
- Failed transactions are not refunded.
- Oracle transactions remain zero-fee.
- Successful non-send transactions refund unused `udo` fee based on `gas_used / gas_limit`.
- Mixed transactions that include a value-fee send refund only the normal gas portion, not the value fee.

Wallets should still simulate before signing and display the submitted fee as the maximum fee. After v23, successful non-send transactions may receive an on-chain `unused_gas_refund` event.
