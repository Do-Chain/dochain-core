# Direct Send Value Fees

DoChain v21 added a consensus-enforced fee rule for direct wallet transfers of native DO.
DoChain v22 extends that rule so direct wallet transfers of other native bank coins can
also pay a value fee, while the required fee coin remains DO (`udo`).

Normal transaction fees still apply to non-send transactions. Oracle vote and prevote transactions remain exempt from normal fee deduction. Governance, staking, smart contract, DEX, module-account, and IBC transactions are not charged the direct-send value fee.

## v21 Parameters

- `enabled`: `true`
- `fee_denom`: `udo`
- `rate_bps`: `1`
- `min_fee`: `1000000000udo`
- `max_fee_enabled`: `false`
- `apply_to_msg_send`: `true`

The direct send fee is:

```text
fee = max(1000000000udo, amount_udo * 1 / 10000)
```

That is a 0.01% fee with a 1000 DO minimum and no maximum cap.

## Wallet Integration

For a transaction that only contains direct user `MsgSend` messages sending `udo`, the wallet should display this direct-send fee as the transaction fee.

For every other transaction type, keep using normal gas estimation and node minimum gas pricing.

Wallets should show:

- transaction type
- amount sent
- direct-send fee, when applicable
- normal gas fee, when applicable
- total fee before signing

## v22 Multi-Denom Rules

v22 keeps all direct-send value fees payable in `udo`.

- `policy_version` must be written by the coordinated upgrade handler before enforcement.
- `udo` is valued directly as DO.
- `udodx` is initially configured at `1 udodx = 1 udo` for fee-value purposes.
- unknown/unconfigured denoms pay the `1000000000udo` minimum on direct wallet sends.
- governance can later add explicit denom value rates for L2 or partner coins.

The v22 direct send fee is:

```text
transfer_value_udo = sum(amount_of_each_known_denom * configured_udo_per_base_unit)
fee_udo = max(1000000000udo, transfer_value_udo * 1 / 10000)
```

If a direct send only contains unknown/unconfigured denoms, the value portion is unknown,
so the chain charges the `1000000000udo` minimum. This keeps unknown coins from becoming
a free spam path without relying on a live price oracle.

Examples with the initial v22 params:

| Transfer | Required value fee |
| --- | ---: |
| `100000000udodx` (100 DODX at 1:1 DO value) | `1000000000udo` |
| `10000000000000000udodx` (10 billion DODX at 1:1 DO value) | `1000000000000udo` |
| any direct send of an unconfigured denom | at least `1000000000udo` |
