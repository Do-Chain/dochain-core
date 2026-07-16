//nolint:revive
package v16

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/Daviddochain/dochain-core/v4/app/keepers"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateV16UpgradeHandler enables DODx staking-backed governance.
func CreateV16UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		keepers.DODxStakingKeeper.SetGovernanceEnabled(sdk.UnwrapSDKContext(ctx), true)
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}
