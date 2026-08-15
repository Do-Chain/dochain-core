//nolint:revive
package v25

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
)

const UpgradeName = "v25"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV25UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
