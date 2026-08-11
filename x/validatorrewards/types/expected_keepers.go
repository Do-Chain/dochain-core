package types

import (
	"context"

	addresscodec "cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
}

type BankKeeper interface {
	HasDenomMetaData(ctx context.Context, denom string) bool
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type StakingKeeper interface {
	ValidatorAddressCodec() addresscodec.Codec
	Validator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.ValidatorI, error)
}

type DistributionKeeper interface {
	AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error
	GetValidatorCurrentRewards(ctx context.Context, val sdk.ValAddress) (distrtypes.ValidatorCurrentRewards, error)
	SetValidatorCurrentRewards(ctx context.Context, val sdk.ValAddress, rewards distrtypes.ValidatorCurrentRewards) error
	GetValidatorOutstandingRewards(ctx context.Context, val sdk.ValAddress) (distrtypes.ValidatorOutstandingRewards, error)
	SetValidatorOutstandingRewards(ctx context.Context, val sdk.ValAddress, rewards distrtypes.ValidatorOutstandingRewards) error
}
