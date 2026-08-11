//nolint:revive
package v19

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
)

const UpgradeName = "v19"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV19UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
