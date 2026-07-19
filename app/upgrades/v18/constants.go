//nolint:revive
package v18

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
)

const UpgradeName = "v18"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV18UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
