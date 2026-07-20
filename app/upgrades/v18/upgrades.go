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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV18UpgradeHandler closes permissionless Wasm upload and default
// instantiation without changing staking, governance, reward, or slashing
// economics.
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

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
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
