package dodxstaking

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/client/cli"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
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

// AppModuleBasic defines the basic DODx staking module.
type AppModuleBasic struct {
	cdc codec.Codec
}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (b AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (AppModuleBasic) DefaultGenesis(codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesisState())
	if err != nil {
		panic(err)
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := json.Unmarshal(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return types.ValidateGenesis(&data)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {}

func (AppModuleBasic) GetTxCmd() *cobra.Command { return cli.GetTxCmd() }

func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

// AppModule implements the DODx staking module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a DODx staking AppModule.
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc},
		keeper:         keeper,
	}
}

func (AppModule) Name() string { return types.ModuleName }

func (AppModule) RegisterInvariants(sdk.InvariantRegistry) {}

func (AppModule) QuerierRoute() string { return types.QuerierRoute }

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))

	if err := cfg.RegisterMigration(types.ModuleName, 1, func(sdk.Context) error { return nil }); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", types.ModuleName, err))
	}
}

func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	if err := json.Unmarshal(data, &genesisState); err != nil {
		panic(err)
	}
	InitGenesis(ctx, am.keeper, &genesisState)
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

func (AppModule) ConsensusVersion() uint64 { return 2 }

func (AppModule) IsAppModule() {}

func (AppModule) IsOnePerModuleType() {}

func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	bz, err := json.MarshalIndent(types.DefaultGenesisState(), "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected default dodxstaking genesis:\n%s\n", bz)
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

func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if am.keeper.RewardsEnabled(sdkCtx) {
		am.keeper.SyncRewardBalances(sdkCtx)
	}
	return nil
}

func (AppModule) EndBlock(context.Context) ([]abci.ValidatorUpdate, error) {
	return []abci.ValidatorUpdate{}, nil
}
