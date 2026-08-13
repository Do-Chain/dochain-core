//nolint:revive
package v23

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
)

const UpgradeName = "v23"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV23UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
