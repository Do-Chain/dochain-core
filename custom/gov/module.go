package gov

import (
	"encoding/json"

	sdkmath "cosmossdk.io/math"
	customtypes "github.com/Daviddochain/dochain-core/v4/custom/gov/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the gov module.
type AppModuleBasic struct {
	gov.AppModuleBasic
}

// NewAppModuleBasic creates a new AppModuleBasic object
func NewAppModuleBasic(proposalHandlers []govclient.ProposalHandler) AppModuleBasic {
	return AppModuleBasic{gov.NewAppModuleBasic(proposalHandlers)}
}

// RegisterLegacyAminoCodec registers the gov module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
	v1.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the gov
// module.
func (am AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	// customize to set default genesis state proposal deposit denom to DODx
	defaultGenesisState := v1.DefaultGenesisState()
	proposalDeposit := sdk.NewCoin(core.MicroDODxDenom, sdkmath.NewInt(core.MicroUnit))
	defaultGenesisState.Params.MinDeposit = sdk.NewCoins(proposalDeposit)
	defaultGenesisState.Params.ExpeditedMinDeposit = sdk.NewCoins(proposalDeposit)

	return cdc.MustMarshalJSON(defaultGenesisState)
}
