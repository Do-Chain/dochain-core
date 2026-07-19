package simulation

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authsimulation "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestRandomizedGenStateCreatesEverySimulationAccount(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	appCodec := codec.NewProtoCodec(interfaceRegistry)
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
		NumBonded:    int64(len(accounts)),
		BondDenom:    sdk.DefaultBondDenom,
		GenTimestamp: time.Unix(1, 0),
	}

	RandomizedGenState(simState, authsimulation.RandomGenesisAccounts)

	var genesis authtypes.GenesisState
	appCodec.MustUnmarshalJSON(simState.GenState[authtypes.ModuleName], &genesis)
	require.Len(t, genesis.Accounts, len(accounts))
	for i, account := range genesis.Accounts {
		var genesisAccount authtypes.GenesisAccount
		require.NoError(t, appCodec.UnpackAny(account, &genesisAccount))
		require.Equal(t, accounts[i].Address, genesisAccount.GetAddress())
	}
}
