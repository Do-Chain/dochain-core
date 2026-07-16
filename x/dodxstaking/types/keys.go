package types

const (
	// ModuleName is the name of the DODx staking module.
	ModuleName = "dodxstaking"

	// StoreKey is the store key for this module.
	StoreKey = ModuleName

	// RouterKey is the message route for this module.
	RouterKey = ModuleName

	// QuerierRoute is the legacy query route for this module.
	QuerierRoute = ModuleName
)

const (
	StakeKeyPrefix byte = 0x01
)

var (
	TotalStakedKey        = []byte{0x02}
	GovernanceEnabledKey  = []byte{0x03}
	GovernanceEnabledFlag = []byte{0x01}
)
