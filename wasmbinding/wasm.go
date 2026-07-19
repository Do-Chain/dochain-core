package wasmbinding

import (
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	customwasm "github.com/Daviddochain/dochain-core/v4/custom/wasm"
	marketkeeper "github.com/Daviddochain/dochain-core/v4/x/market/keeper"
	markettypes "github.com/Daviddochain/dochain-core/v4/x/market/types"
	oraclekeeper "github.com/Daviddochain/dochain-core/v4/x/oracle/keeper"
	treasurykeeper "github.com/Daviddochain/dochain-core/v4/x/treasury/keeper"
	treasurytypes "github.com/Daviddochain/dochain-core/v4/x/treasury/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
)

func RegisterCustomPlugins(
	marketKeeper *marketkeeper.Keeper,
	oracleKeeper *oraclekeeper.Keeper,
	treasuryKeeper *treasurykeeper.Keeper,
) []wasmkeeper.Option {
	wasmQueryPlugin := NewQueryPlugin(
		marketKeeper,
		oracleKeeper,
		treasuryKeeper,
	)

	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})
	messengerDecoratorOpt := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(marketKeeper),
	)

	return []wasmkeeper.Option{
		queryPluginOpt,
		messengerDecoratorOpt,
	}
}

func RegisterStargateQueries(queryRouter baseapp.GRPCQueryRouter, codec codec.Codec) []wasmkeeper.Option {
	return RegisterStargateQueriesWithKeepers(queryRouter, codec, nil, nil)
}

func RegisterStargateQueriesWithKeepers(
	queryRouter baseapp.GRPCQueryRouter,
	codec codec.Codec,
	marketKeeper *marketkeeper.Keeper,
	treasuryKeeper *treasurykeeper.Keeper,
) []wasmkeeper.Option {
	var marketQueryServer markettypes.QueryServer
	if marketKeeper != nil {
		marketQueryServer = marketkeeper.NewQuerier(*marketKeeper)
	}
	var treasuryQueryServer treasurytypes.QueryServer
	if treasuryKeeper != nil {
		treasuryQueryServer = treasurykeeper.NewQuerier(*treasuryKeeper)
	}

	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: stargateQuerierWithQueryServers(queryRouter, codec, marketQueryServer, treasuryQueryServer),
	})

	return []wasmkeeper.Option{
		queryPluginOpt,
	}
}

// RegisterLegacyQueryHandler wraps the wasm query handler with legacy store support for historical queries
func RegisterLegacyQueryHandler(storeKey storetypes.StoreKey) wasmkeeper.Option {
	return wasmkeeper.WithQueryHandlerDecorator(func(next wasmkeeper.WasmVMQueryHandler) wasmkeeper.WasmVMQueryHandler {
		return customwasm.NewLegacyQueryHandler(next, storeKey)
	})
}
