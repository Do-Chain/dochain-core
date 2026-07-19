package types

const (
	EventTypeStake          = "dodx_stake"
	EventTypeUnstake        = "dodx_unstake"
	EventTypeDepositRewards = "dodx_deposit_rewards"
	EventTypeClaimRewards   = "dodx_claim_rewards"
	EventTypeSyncRewards    = "dodx_sync_rewards"

	AttributeKeyStaker    = "staker"
	AttributeKeyClaimer   = "claimer"
	AttributeKeyDepositor = "depositor"
	AttributeKeyAmount    = "amount"
	AttributeKeyDenom     = "denom"
)
