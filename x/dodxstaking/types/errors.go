package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidAmount       = errorsmod.Register(ModuleName, 2, "invalid DODx staking amount")
	ErrInsufficientStake   = errorsmod.Register(ModuleName, 3, "insufficient staked DODx")
	ErrInvalidStakeDenom   = errorsmod.Register(ModuleName, 4, "DODx staking only accepts udodx")
	ErrDuplicateStake      = errorsmod.Register(ModuleName, 5, "duplicate DODx stake record")
	ErrGovernanceDisabled  = errorsmod.Register(ModuleName, 6, "DODx governance staking is not enabled")
	ErrInvalidRewardDenom  = errorsmod.Register(ModuleName, 7, "invalid DODx staking reward denom")
	ErrInvalidRewardAmount = errorsmod.Register(ModuleName, 8, "invalid DODx staking reward amount")
	ErrNoRewards           = errorsmod.Register(ModuleName, 9, "no DODx staking rewards available")
)
