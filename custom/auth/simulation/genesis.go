package simulation

import (
	"github.com/cosmos/cosmos-sdk/types/module"
	authsimulation "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// RandomizedGenState generates a random GenesisState for auth
func RandomizedGenState(simState *module.SimulationState, randGenAccountsFn authtypes.RandomGenesisAccountsFn) {
	authsimulation.RandomizedGenState(simState, randGenAccountsFn)
}
