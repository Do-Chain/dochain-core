package types

import (
	"bytes"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestRegisterInterfacesIncludesStandardAndLazyVestingAccounts(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	RegisterInterfaces(registry)

	coins := sdk.NewCoins(sdk.NewInt64Coin("udo", 10))
	newBaseAccount := func(marker byte) *authtypes.BaseAccount {
		return authtypes.NewBaseAccountWithAddress(sdk.AccAddress(bytes.Repeat([]byte{marker}, 20)))
	}
	baseVesting, err := vestingtypes.NewBaseVestingAccount(newBaseAccount(1), coins, 10)
	require.NoError(t, err)
	continuous, err := vestingtypes.NewContinuousVestingAccount(newBaseAccount(2), coins, 1, 10)
	require.NoError(t, err)
	delayed, err := vestingtypes.NewDelayedVestingAccount(newBaseAccount(3), coins, 10)
	require.NoError(t, err)
	periodic, err := vestingtypes.NewPeriodicVestingAccount(
		newBaseAccount(4),
		coins,
		0,
		vestingtypes.Periods{{Length: 10, Amount: coins}},
	)
	require.NoError(t, err)
	permanent, err := vestingtypes.NewPermanentLockedAccount(newBaseAccount(5), coins)
	require.NoError(t, err)
	lazy := NewLazyGradedVestingAccount(newBaseAccount(6), coins, nil)

	testCases := []struct {
		typeURL string
		account proto.Message
		vesting bool
	}{
		{"/cosmos.vesting.v1beta1.BaseVestingAccount", baseVesting, false},
		{"/cosmos.vesting.v1beta1.ContinuousVestingAccount", continuous, true},
		{"/cosmos.vesting.v1beta1.DelayedVestingAccount", delayed, true},
		{"/cosmos.vesting.v1beta1.PeriodicVestingAccount", periodic, true},
		{"/cosmos.vesting.v1beta1.PermanentLockedAccount", permanent, true},
		{"/do.vesting.v1beta1.LazyGradedVestingAccount", lazy, true},
	}

	for _, tc := range testCases {
		t.Run(tc.typeURL, func(t *testing.T) {
			resolved, err := registry.Resolve(tc.typeURL)
			require.NoError(t, err)
			require.IsType(t, tc.account, resolved)

			packed, err := codectypes.NewAnyWithValue(tc.account)
			require.NoError(t, err)
			var account sdk.AccountI
			require.NoError(t, registry.UnpackAny(packed, &account))
			require.IsType(t, tc.account, account)

			var genesisAccount authtypes.GenesisAccount
			require.NoError(t, registry.UnpackAny(packed, &genesisAccount))
			require.IsType(t, tc.account, genesisAccount)

			if tc.vesting {
				var vestingAccount vestingexported.VestingAccount
				require.NoError(t, registry.UnpackAny(packed, &vestingAccount))
				require.IsType(t, tc.account, vestingAccount)
			}
		})
	}
}
