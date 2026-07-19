package types

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
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
	rewardDenoms := map[string]bool{}

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

	if err := validateRewardAmountRecords(data.RewardAccumulators, "reward accumulator"); err != nil {
		return err
	}
	if err := validateRewardAmountRecords(data.RewardPools, "reward pool"); err != nil {
		return err
	}
	if err := validateAccountRewardAmountRecords(data.RewardDebts, "reward debt"); err != nil {
		return err
	}
	if err := validateAccountRewardAmountRecords(data.PendingRewards, "pending reward"); err != nil {
		return err
	}
	for _, record := range data.RewardAccumulators {
		rewardDenoms[record.Denom] = true
	}
	for _, record := range data.RewardPools {
		rewardDenoms[record.Denom] = true
	}
	for _, record := range data.RewardDebts {
		rewardDenoms[record.Denom] = true
	}
	for _, record := range data.PendingRewards {
		rewardDenoms[record.Denom] = true
	}
	if len(rewardDenoms) > MaxRewardDenoms {
		return fmt.Errorf("too many reward denoms: %d > %d", len(rewardDenoms), MaxRewardDenoms)
	}

	return nil
}

// GetGenesisStateFromAppState returns x/dodxstaking GenesisState from app genesis.
func GetGenesisStateFromAppState(_ codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState
	if appState[ModuleName] != nil {
		if err := json.Unmarshal(appState[ModuleName], &genesisState); err != nil {
			panic(err)
		}
	}
	return &genesisState
}

func validateRewardAmountRecords(records []RewardAmountRecord, label string) error {
	seen := map[string]bool{}
	for _, record := range records {
		if err := validateRewardDenom(record.Denom); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if seen[record.Denom] {
			return fmt.Errorf("%s: duplicate denom %s", label, record.Denom)
		}
		seen[record.Denom] = true
		if _, err := ParsePositiveGenesisInt(record.Amount, label); err != nil {
			return err
		}
	}
	return nil
}

func validateAccountRewardAmountRecords(records []AccountRewardAmountRecord, label string) error {
	seen := map[string]bool{}
	for _, record := range records {
		addr, err := sdk.AccAddressFromBech32(record.Address)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if err := validateRewardDenom(record.Denom); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		key := addr.String() + "/" + record.Denom
		if seen[key] {
			return fmt.Errorf("%s: duplicate account denom %s", label, key)
		}
		seen[key] = true
		if _, err := ParsePositiveGenesisInt(record.Amount, label); err != nil {
			return err
		}
	}
	return nil
}

// ParsePositiveGenesisInt parses a positive integer amount stored in module genesis.
func ParsePositiveGenesisInt(value, label string) (sdkmath.Int, error) {
	amount, ok := sdkmath.NewIntFromString(value)
	if !ok || !amount.IsPositive() {
		return sdkmath.ZeroInt(), fmt.Errorf("%s: invalid positive integer amount %q", label, value)
	}
	return amount, nil
}
