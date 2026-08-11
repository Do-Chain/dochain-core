//nolint:revive
package v19

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	validatorrewardstypes "github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
)

const UpgradeName = "v19"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV19UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{validatorrewardstypes.StoreKey},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
