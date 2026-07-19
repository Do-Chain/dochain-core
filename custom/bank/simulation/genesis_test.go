package simulation

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	appparams "github.com/Daviddochain/dochain-core/v4/app/params"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestRandomizedGenStateUsesSingleDODenomination(t *testing.T) {
	appCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	accounts := []simtypes.Account{
		{Address: sdk.AccAddress(bytes.Repeat([]byte{1}, 20))},
		{Address: sdk.AccAddress(bytes.Repeat([]byte{2}, 20))},
	}
	simState := &module.SimulationState{
		AppParams:    simtypes.AppParams{},
		Cdc:          appCodec,
		Rand:         rand.New(rand.NewSource(1)),
		GenState:     map[string]json.RawMessage{},
		Accounts:     accounts,
		InitialStake: math.NewInt(100),
		NumBonded:    1,
		BondDenom:    appparams.BondDenom,
	}

	require.NotPanics(t, func() { RandomizedGenState(simState) })

	var genesis banktypes.GenesisState
	appCodec.MustUnmarshalJSON(simState.GenState[banktypes.ModuleName], &genesis)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(appparams.BondDenom, 300)), genesis.Supply)
	require.Len(t, genesis.Balances, len(accounts))
	for _, balance := range genesis.Balances {
		require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(appparams.BondDenom, 100)), balance.Coins)
	}
	require.Equal(t, []banktypes.SendEnabled{{Denom: appparams.BondDenom, Enabled: true}}, genesis.SendEnabled)
}
