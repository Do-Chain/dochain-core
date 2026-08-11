# Validator Reward Campaigns

`x/validatorrewards` lets an approved token admin fund rewards for delegators of selected validators. It is intended for Do-wallet validator accounts and partner/L2 token campaigns.

## Security Model

- Campaign denoms must be registered by chain authority/governance before use.
- The denom must be a valid bank denom with on-chain bank metadata.
- Only the registered denom admin can create campaigns for that denom.
- There is no minimum deposit amount.
- Rewards are held by the `validatorrewards` module until released.
- Drip schedules are released by permissionless `MsgReleaseCampaign`; the module does not scan every campaign in BeginBlock.

## Reward Modes

`delegators_only = false`

Uses standard `x/distribution` validator reward allocation. Validator commission applies.

`delegators_only = true`

Credits validator current/outstanding rewards directly for delegators and does not accrue validator commission.

## Wallet-Facing Messages

Register a reward denom admin through governance/authority:

```json
{
  "@type": "/do.validatorrewards.v1beta1.MsgRegisterRewardDenom",
  "authority": "<gov-module-address>",
  "denom": "ul2coin",
  "admin": "do1..."
}
```

Create a campaign:

```json
{
  "@type": "/do.validatorrewards.v1beta1.MsgCreateCampaign",
  "admin": "do1...",
  "denom": "ul2coin",
  "title": "L2 Coin validator rewards",
  "description": "100B per selected validator",
  "allocations": [
    {
      "validator_address": "dovaloper1...",
      "amount": { "denom": "ul2coin", "amount": "100000000000000000" },
      "released": "0"
    }
  ],
  "start_height": "0",
  "end_height": "0",
  "delegators_only": true
}
```

Release vested rewards:

```json
{
  "@type": "/do.validatorrewards.v1beta1.MsgReleaseCampaign",
  "releaser": "do1...",
  "campaign_id": "1"
}
```

Cancel unreleased rewards:

```json
{
  "@type": "/do.validatorrewards.v1beta1.MsgCancelCampaign",
  "admin": "do1...",
  "campaign_id": "1"
}
```

## Do-Wallet Tab

For a validator/admin wallet, add a "Validator Rewards" tab that uses:

- `QueryRewardDenoms` to list denoms the wallet can administer.
- `QueryCampaigns` or `QueryValidatorCampaigns` to show campaign status.
- `MsgCreateCampaign` with one or more validator allocations for batch funding.
- `MsgReleaseCampaign` when vested rewards are available.
- Existing `cosmos.distribution.v1beta1.Query/DelegationRewards` for delegator claimable rewards.

For immediate 100B DO per validator, use `end_height = 0`. For a drip campaign, set `start_height` and `end_height`; released amount is linear by block height.
