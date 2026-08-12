package v22

import (
	"testing"

	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	"github.com/stretchr/testify/require"
)

func TestV22HasNoStoreMigration(t *testing.T) {
	require.Empty(t, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, Upgrade.StoreUpgrades.Renamed)
}

func TestV22ValueFeeParams(t *testing.T) {
	params := valuefeetypes.MainnetV22Params()

	require.NoError(t, params.Validate())
	require.True(t, params.Enabled)
	require.True(t, params.ChargeUnknownDenomMinFee)
	require.Equal(t, uint32(22), params.PolicyVersion)
	require.Len(t, params.DenomValueRates, 2)
}
