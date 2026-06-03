package types

const (
	ModuleName   = "mfa"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var PolicyKeyPrefix = []byte{0x01}

func PolicyKey(account string) []byte {
	key := make([]byte, 0, len(PolicyKeyPrefix)+len(account))
	key = append(key, PolicyKeyPrefix...)
	key = append(key, []byte(account)...)
	return key
}
