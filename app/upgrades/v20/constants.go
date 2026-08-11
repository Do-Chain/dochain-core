//nolint:revive
package v20

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	validatorrewardstypes "github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
)

const UpgradeName = "v20"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV20UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{validatorrewardstypes.StoreKey},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
