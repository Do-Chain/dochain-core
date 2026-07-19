package types

const (
	// ModuleName is the name of the DODx staking module.
	ModuleName = "dodxstaking"

	// StoreKey is the store key for this module.
	StoreKey = ModuleName

	// RouterKey is the message route for this module.
	RouterKey = ModuleName

	// QuerierRoute is the legacy query route for this module.
	QuerierRoute = ModuleName
)

const (
	StakeKeyPrefix byte = 0x01
	// RewardAccumulatorKeyPrefix stores cumulative reward-per-staked-DODx by reward denom.
	RewardAccumulatorKeyPrefix byte = 0x10
	// RewardPoolKeyPrefix stores total unclaimed/accounted rewards by reward denom.
	RewardPoolKeyPrefix byte = 0x11
	// RewardDebtKeyPrefix stores per-account settled accumulator debt by reward denom.
	RewardDebtKeyPrefix byte = 0x12
	// PendingRewardKeyPrefix stores per-account claimable rewards by reward denom.
	PendingRewardKeyPrefix byte = 0x13
	// RewardDenomKeyPrefix indexes reward denoms that have ever been accounted.
	RewardDenomKeyPrefix byte = 0x14
)

var (
	TotalStakedKey        = []byte{0x02}
	GovernanceEnabledKey  = []byte{0x03}
	GovernanceEnabledFlag = []byte{0x01}
	RewardsEnabledKey     = []byte{0x04}
	RewardsEnabledFlag    = []byte{0x01}
)
