# DoChain Gas Fee Policy v23

v23 keeps direct wallet-send value fees and normal transaction gas as separate paths.

## Direct wallet sends

Direct user `MsgSend` transactions for DO, DODX, or unknown bank denoms use the value-send fee only:

```text
value_fee = max(amount_value_in_udo * rate_bps / 10000, min_fee_udo)
```

Mainnet v23 defaults:

```text
rate_bps = 1
min_fee_udo = 1000000000
fee_denom = udo
```

That is 0.01% with a 1,000 DO minimum, paid in DO/`udo`.

## Other transactions

Non-send transactions use normal gas:

```text
required_fee = gas_limit * normal_gas_price
```

Mainnet v23 default:

```text
normal_gas_price = 10000000udo
```

Wallets should simulate the transaction, apply an adjustment, then set the fee from the chain gas policy instead of hardcoding a fee per message type.

## Successful unused gas refund

v23 refunds unused normal gas for successful non-send transactions:

```text
gas_used_fee = gas_used * normal_gas_price
refund = gas_limit_fee - gas_used_fee
```

The refund is sent from the fee collector module account back to the fee payer after successful message execution.

Pure direct value-send transactions are excluded from this refund because they do not pay normal gas under the v22/v23 send policy.

Failed transactions are not refunded by this post-handler. Cosmos SDK discards message and post-handler state for failed message execution, so failed-transaction refunds require a separate BaseApp-level design.

## Chain-served config

The canonical fee policy is stored in the `valuefee` params subspace. Clients can read the params through the standard params query API, including:

```text
NormalGasPrices
RefundUnusedGas
RateBps
MinFee
FeeDenom
DenomValueRates
ChargeUnknownDenomMinFee
PolicyVersion
```

Validators should keep local `minimum-gas-prices` aligned with `NormalGasPrices` so mempool admission matches the chain's DeliverTx fee checks.

## Dynamic gas

v23 adds explicit dynamic gas policy fields, but `DynamicGasEnabled` is intentionally rejected until a real base-fee updater is implemented and tested. This prevents governance or a bad config from enabling a half-wired fee market.
