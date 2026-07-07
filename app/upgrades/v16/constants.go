//nolint:revive
package v16

import (
	store "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	dodxstakingtypes "github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
)

const UpgradeName = "v16"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateV16UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{dodxstakingtypes.StoreKey},
		Deleted: []string{},
		Renamed: []store.StoreRename{},
	},
}
