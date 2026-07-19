package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/gogoproto/proto"
)

// RegisterLegacyAminoCodec registers the account interfaces and concrete types on the
// provided LegacyAmino codec. These types are used for Amino JSON serialization
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterInterface((*exported.VestingAccount)(nil), nil)
	cdc.RegisterConcrete(&vestingtypes.BaseVestingAccount{}, "core/BaseVestingAccount", nil)
	cdc.RegisterConcrete(&LazyGradedVestingAccount{}, "core/LazyGradedVestingAccount", nil)
}

// RegisterInterfaces associates protoName with AccountI and VestingAccount
// Interfaces and creates a registry of it's concrete implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterInterface(
		"cosmos.vesting.v1beta1.VestingAccount",
		(*exported.VestingAccount)(nil),
		&vestingtypes.ContinuousVestingAccount{},
		&vestingtypes.DelayedVestingAccount{},
		&vestingtypes.PeriodicVestingAccount{},
		&vestingtypes.PermanentLockedAccount{},
		&LazyGradedVestingAccount{},
	)

	vestingAccounts := []proto.Message{
		&vestingtypes.BaseVestingAccount{},
		&vestingtypes.DelayedVestingAccount{},
		&vestingtypes.ContinuousVestingAccount{},
		&vestingtypes.PeriodicVestingAccount{},
		&vestingtypes.PermanentLockedAccount{},
		&LazyGradedVestingAccount{},
	}

	registry.RegisterImplementations(
		(*authtypes.AccountI)(nil),
		vestingAccounts...,
	)

	registry.RegisterImplementations(
		(*sdk.AccountI)(nil),
		vestingAccounts...,
	)

	registry.RegisterImplementations(
		(*authtypes.GenesisAccount)(nil),
		vestingAccounts...,
	)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/oracle module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding as Amino is
	// still used for that purpose.
	//
	// The actual codec used for serialization should be provided to x/staking and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
