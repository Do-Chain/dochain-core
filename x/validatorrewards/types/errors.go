package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidDenom        = errorsmod.Register(ModuleName, 2, "invalid validator reward denom")
	ErrRewardDenomMissing  = errorsmod.Register(ModuleName, 3, "validator reward denom is not registered")
	ErrUnauthorizedAdmin   = errorsmod.Register(ModuleName, 4, "unauthorized validator reward denom admin")
	ErrInvalidCampaign     = errorsmod.Register(ModuleName, 5, "invalid validator reward campaign")
	ErrCampaignNotFound    = errorsmod.Register(ModuleName, 6, "validator reward campaign not found")
	ErrCampaignInactive    = errorsmod.Register(ModuleName, 7, "validator reward campaign is inactive")
	ErrNoReleasableRewards = errorsmod.Register(ModuleName, 8, "no validator reward campaign rewards are releasable")
)
