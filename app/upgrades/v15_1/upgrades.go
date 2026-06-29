//nolint:revive
package v15_1

import (
	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/Daviddochain/dochain-core/v4/app/keepers"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const doChainID = "Do-Chain"

var cosmWasmUploadAllowlist = []string{}

func runForkLogic(ctx sdk.Context, appKeepers *keepers.AppKeepers, _ *module.Manager) {
	if ctx.ChainID() != doChainID {
		return
	}

	setGovDepositParams(ctx, appKeepers)
	setWasmAccessParams(ctx, appKeepers)
	disableDelegatorSlashingOnJail(ctx, appKeepers)
}

func setGovDepositParams(ctx sdk.Context, appKeepers *keepers.AppKeepers) {
	params, err := appKeepers.GovKeeper.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	proposalDeposit := sdk.NewCoin(core.MicroDODxDenom, sdkmath.NewInt(core.MicroUnit))
	params.MinDeposit = sdk.NewCoins(proposalDeposit)
	params.ExpeditedMinDeposit = sdk.NewCoins(proposalDeposit)

	if err := appKeepers.GovKeeper.Params.Set(ctx, params); err != nil {
		panic(err)
	}
}

func setWasmAccessParams(ctx sdk.Context, appKeepers *keepers.AppKeepers) {
	params := appKeepers.WasmKeeper.GetParams(ctx)
	params.CodeUploadAccess = wasmUploadAccessConfig()
	params.InstantiateDefaultPermission = params.CodeUploadAccess.Permission

	if err := params.ValidateBasic(); err != nil {
		panic(err)
	}
	if err := appKeepers.WasmKeeper.SetParams(ctx, params); err != nil {
		panic(err)
	}
}

func wasmUploadAccessConfig() wasmtypes.AccessConfig {
	if len(cosmWasmUploadAllowlist) > 0 {
		return wasmtypes.AccessConfig{
			Permission: wasmtypes.AccessTypeAnyOfAddresses,
			Addresses:  cosmWasmUploadAllowlist,
		}
	}

	return wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody}
}

func disableDelegatorSlashingOnJail(ctx sdk.Context, appKeepers *keepers.AppKeepers) {
	oracleParams := appKeepers.OracleKeeper.GetParams(ctx)
	oracleParams.SlashFraction = sdkmath.LegacyZeroDec()
	appKeepers.OracleKeeper.SetParams(ctx, oracleParams)

	slashingParams, err := appKeepers.SlashingKeeper.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	slashingParams.SlashFractionDowntime = sdkmath.LegacyZeroDec()
	slashingParams.SlashFractionDoubleSign = sdkmath.LegacyZeroDec()
	if err := appKeepers.SlashingKeeper.SetParams(ctx, slashingParams); err != nil {
		panic(err)
	}
}
