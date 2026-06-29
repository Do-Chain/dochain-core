//nolint:revive
package v15_1

import (
	"github.com/Daviddochain/dochain-core/v4/app/upgrades"
	"github.com/Daviddochain/dochain-core/v4/types/fork"
)

const UpgradeName = "v15_1"

var Fork = upgrades.Fork{
	UpgradeName:    UpgradeName,
	UpgradeHeight:  fork.DoCommunityGovernanceHeight,
	BeginForkLogic: runForkLogic,
}
