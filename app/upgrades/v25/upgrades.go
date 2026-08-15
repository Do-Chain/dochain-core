//nolint:revive
package v25

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/Daviddochain/dochain-core/v4/app/keepers"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateV25UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		params := valuefeetypes.MainnetV25Params()
		if err := params.Validate(); err != nil {
			return nil, err
		}
		keepers.ValueFeeKeeper.SetParams(sdkCtx, params)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
