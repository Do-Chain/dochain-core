//nolint:revive
package v20

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/Daviddochain/dochain-core/v4/app/keepers"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const dodexWasmUploadAdmin = "do1t7rnyus7q3667txrwexcrgkjpr5rwm8z0qaycg"

// CreateV20UpgradeHandler narrows Wasm uploads to the controlled DODEX admin
// wallet and keeps default instantiation closed.
func CreateV20UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		params := whitelistDodexWasmUploader(keepers.WasmKeeper.GetParams(ctx))
		if err := params.ValidateBasic(); err != nil {
			return nil, err
		}
		if err := keepers.WasmKeeper.SetParams(ctx, params); err != nil {
			return nil, err
		}
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

func whitelistDodexWasmUploader(params wasmtypes.Params) wasmtypes.Params {
	params.CodeUploadAccess.Permission = wasmtypes.AccessTypeAnyOfAddresses
	params.CodeUploadAccess.Addresses = appendUniqueAddress(params.CodeUploadAccess.Addresses, dodexWasmUploadAdmin)
	params.InstantiateDefaultPermission = wasmtypes.AccessTypeNobody
	return params
}

func appendUniqueAddress(addresses []string, address string) []string {
	for _, existing := range addresses {
		if existing == address {
			return addresses
		}
	}
	return append(addresses, address)
}
