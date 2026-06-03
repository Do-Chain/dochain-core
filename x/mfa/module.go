package mfa

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/Daviddochain/dochain-core/v4/x/mfa/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/mfa/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

func (AppModuleBasic) RegisterInterfaces(codectypes.InterfaceRegistry) {}

func (AppModuleBasic) DefaultGenesis(codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesisState())
	if err != nil {
		panic(err)
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if len(bz) == 0 {
		gs = types.DefaultGenesisState()
	} else if err := json.Unmarshal(bz, &gs); err != nil {
		return err
	}
	return types.ValidateGenesis(gs)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {}

func (AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{keeper: k}
}

func (AppModule) Name() string { return types.ModuleName }

func (AppModule) RegisterInvariants(sdk.InvariantRegistry) {}

func (AppModule) QuerierRoute() string { return types.QuerierRoute }

func (AppModule) RegisterServices(module.Configurator) {}

func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, bz json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	if len(bz) == 0 {
		gs = types.DefaultGenesisState()
	} else if err := json.Unmarshal(bz, &gs); err != nil {
		panic(err)
	}
	InitGenesis(ctx, am.keeper, gs)
	return nil
}

func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	return bz
}

func (AppModule) ConsensusVersion() uint64 { return 1 }

func (AppModule) IsAppModule() {}

func (AppModule) IsOnePerModuleType() {}

func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	bz, err := json.MarshalIndent(types.DefaultGenesisState(), "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected default mfa parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = bz
}

func (AppModule) ProposalContents(module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

func (AppModule) RandomizedParams(*rand.Rand) []simtypes.LegacyParamChange {
	return nil
}

func (AppModule) RegisterStoreDecoder(simtypes.StoreDecoderRegistry) {}

func (AppModule) WeightedOperations(module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

func (AppModule) BeginBlock(context.Context) error { return nil }

func (AppModule) EndBlock(context.Context) ([]abci.ValidatorUpdate, error) {
	return []abci.ValidatorUpdate{}, nil
}
