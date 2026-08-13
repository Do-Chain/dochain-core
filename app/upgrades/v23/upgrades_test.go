package v23

import (
	"testing"

	core "github.com/Daviddochain/dochain-core/v4/types"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	"github.com/stretchr/testify/require"
)

func TestV23HasNoStoreMigration(t *testing.T) {
	require.Empty(t, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, Upgrade.StoreUpgrades.Renamed)
}

func TestV23GasPolicyParams(t *testing.T) {
	params := valuefeetypes.MainnetV23Params()

	require.NoError(t, params.Validate())
	require.True(t, params.Enabled)
	require.True(t, params.ChargeUnknownDenomMinFee)
	require.True(t, params.RefundUnusedGas)
	require.False(t, params.DynamicGasEnabled)
	require.Equal(t, uint32(23), params.PolicyVersion)
	require.Equal(t, "10000000.000000000000000000udo", params.NormalGasPrices.String())
	require.Equal(t, core.MicroDoDenom, params.FeeDenom)
}
