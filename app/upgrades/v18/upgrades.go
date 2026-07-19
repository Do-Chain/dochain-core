//nolint:revive
package v18

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/Daviddochain/dochain-core/v4/app/keepers"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
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

		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

func secureWasmAccessParams(params wasmtypes.Params) wasmtypes.Params {
	params.CodeUploadAccess = wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeNobody}
	params.InstantiateDefaultPermission = wasmtypes.AccessTypeNobody
	return params
}
