# Direct DO Send Fees

DoChain v21 adds a consensus-enforced fee rule for direct wallet transfers of native DO.

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
