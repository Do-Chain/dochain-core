package gov

import (
	"encoding/json"
	"time"

	sdkmath "cosmossdk.io/math"
	customtypes "github.com/Daviddochain/dochain-core/v4/custom/gov/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/cosmos/cosmos-sdk/codec"
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
	// customize to set default genesis state deposit denom to udo
	defaultGenesisState := v1.DefaultGenesisState()
	validatorStage := 48 * time.Hour
	publicStage := 5 * 24 * time.Hour

	defaultGenesisState.Params.MinDeposit[0].Denom = core.MicroDoDenom
	defaultGenesisState.Params.MaxDepositPeriod = &validatorStage
	defaultGenesisState.Params.VotingPeriod = &publicStage
	defaultGenesisState.Params.ExpeditedVotingPeriod = &publicStage
	defaultGenesisState.Params.Quorum = sdkmath.LegacyZeroDec().String()
	defaultGenesisState.Params.Threshold = sdkmath.LegacyNewDecWithPrec(5, 1).String()
	defaultGenesisState.Params.ExpeditedThreshold = defaultGenesisState.Params.Threshold
	defaultGenesisState.Params.VetoThreshold = sdkmath.LegacyOneDec().String()
	defaultGenesisState.Params.BurnVoteQuorum = false
	defaultGenesisState.Params.BurnVoteVeto = false
	defaultGenesisState.Params.BurnProposalDepositPrevote = false
	if len(defaultGenesisState.Params.ExpeditedMinDeposit) > 0 {
		defaultGenesisState.Params.ExpeditedMinDeposit[0].Denom = core.MicroDoDenom
	}

	return cdc.MustMarshalJSON(defaultGenesisState)
}
