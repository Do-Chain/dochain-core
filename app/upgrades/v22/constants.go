//nolint:revive
package v22

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
)

const UpgradeName = "v22"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV22UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
