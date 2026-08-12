//nolint:revive
package v18

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/Daviddochain/dochain-core/v4/app/keepers"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	core "github.com/Daviddochain/dochain-core/v4/types"
	oracletypes "github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV18UpgradeHandler closes permissionless Wasm upload and default
// instantiation, and enables native DEX udo reward accounting for DODX stakers.
func CreateV18UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		params := secureWasmAccessParams(keepers.WasmKeeper.GetParams(ctx))
		if err := params.ValidateBasic(); err != nil {
			return nil, err
		}
		if err := keepers.WasmKeeper.SetParams(ctx, params); err != nil {
			return nil, err
		}
		if err := secureExistingWasmInstantiatePermissions(sdk.UnwrapSDKContext(ctx), keepers.WasmKeeper); err != nil {
			return nil, err
		}
		if err := restoreOraclePenaltyParams(ctx, keepers); err != nil {
			return nil, err
		}
		if err := enableNativeDodxRewardCollection(sdk.UnwrapSDKContext(ctx), keepers); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

func enableNativeDodxRewardCollection(ctx sdk.Context, keepers *keepers.AppKeepers) error {
	keepers.DODxStakingKeeper.SetRewardsEnabled(ctx, true)
	if err := keepers.DODxStakingKeeper.RegisterRewardDenom(ctx, core.MicroDoDenom); err != nil {
		return fmt.Errorf("register DODX staking reward denom %s: %w", core.MicroDoDenom, err)
	}
	keepers.DODxStakingKeeper.SyncRewardBalances(ctx)
	return nil
}

func secureWasmAccessParams(params wasmtypes.Params) wasmtypes.Params {
	params.CodeUploadAccess = wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeNobody}
	params.InstantiateDefaultPermission = wasmtypes.AccessTypeNobody
	return params
}

func secureExistingWasmInstantiatePermissions(ctx sdk.Context, wasmKeeper wasmkeeper.Keeper) error {
	codeIDs := make([]uint64, 0)
	wasmKeeper.IterateCodeInfos(ctx, func(codeID uint64, info wasmtypes.CodeInfo) bool {
		if shouldSecureWasmInstantiateConfig(info) {
			codeIDs = append(codeIDs, codeID)
		}
		return false
	})

	govWasmKeeper := wasmkeeper.NewGovPermissionKeeper(wasmKeeper)
	var emptyCaller sdk.AccAddress
	for _, codeID := range codeIDs {
		if err := govWasmKeeper.SetAccessConfig(ctx, codeID, emptyCaller, wasmtypes.AllowNobody); err != nil {
			return fmt.Errorf("secure wasm instantiate permission for code id %d: %w", codeID, err)
		}
	}
	return nil
}

func shouldSecureWasmInstantiateConfig(info wasmtypes.CodeInfo) bool {
	return !info.InstantiateConfig.Equals(wasmtypes.AllowNobody)
}

func restoreOraclePenaltyParams(ctx context.Context, keepers *keepers.AppKeepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	oracleParams := secureOraclePenaltyParams(keepers.OracleKeeper.GetParams(sdkCtx))
	if err := oracleParams.Validate(); err != nil {
		return fmt.Errorf("secure oracle penalty params: %w", err)
	}
	keepers.OracleKeeper.SetParams(sdkCtx, oracleParams)

	return nil
}

func secureOraclePenaltyParams(params oracletypes.Params) oracletypes.Params {
	if params.SlashFraction.IsZero() {
		params.SlashFraction = oracletypes.DefaultSlashFraction
	}
	if params.MinValidPerWindow.IsZero() {
		params.MinValidPerWindow = oracletypes.DefaultMinValidPerWindow
	}
	return params
}
