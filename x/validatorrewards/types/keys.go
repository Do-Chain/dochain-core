package types

const (
	ModuleName = "validatorrewards"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	QuerierRoute = ModuleName
)

const (
	RewardDenomKeyPrefix       byte = 0x01
	CampaignKeyPrefix          byte = 0x02
	ValidatorCampaignKeyPrefix byte = 0x03
)

var NextCampaignIDKey = []byte{0x04}
