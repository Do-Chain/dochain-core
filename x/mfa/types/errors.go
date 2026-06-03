package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrMFARequired        = errorsmod.Register(ModuleName, 2, "mfa approval required")
	ErrInvalidMFAApproval = errorsmod.Register(ModuleName, 3, "invalid mfa approval")
	ErrExpiredMFAApproval = errorsmod.Register(ModuleName, 4, "expired mfa approval")
	ErrInvalidMFAPolicy   = errorsmod.Register(ModuleName, 5, "invalid mfa policy")
)
