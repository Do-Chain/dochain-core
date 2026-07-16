package types

import (
	"encoding/json"

	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

// ValidateGenesis validates x/dodxstaking genesis state.
func ValidateGenesis(data *GenesisState) error {
	seen := map[string]bool{}

	for _, stake := range data.Stakes {
		addr, err := sdk.AccAddressFromBech32(stake.Address)
		if err != nil {
			return err
		}

		key := addr.String()
		if seen[key] {
			return ErrDuplicateStake
		}
		seen[key] = true

		if stake.Amount.Denom != core.MicroDODxDenom {
			return ErrInvalidStakeDenom
		}
		if !stake.Amount.IsValid() || !stake.Amount.IsPositive() {
			return ErrInvalidAmount
		}
	}

	return nil
}

// GetGenesisStateFromAppState returns x/dodxstaking GenesisState from app genesis.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState
	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}
	return &genesisState
}
