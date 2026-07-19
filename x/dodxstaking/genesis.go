package dodxstaking

import (
	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes x/dodxstaking state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, data *types.GenesisState) {
	for _, stake := range data.Stakes {
		addr, err := sdk.AccAddressFromBech32(stake.Address)
		if err != nil {
			panic(err)
		}
		k.SetStakeAmount(ctx, addr, stake.Amount.Amount)
		k.SetTotalStakedAmount(ctx, k.GetTotalStakedAmount(ctx).Add(stake.Amount.Amount))
	}

	for _, record := range data.RewardAccumulators {
		amount, err := types.ParsePositiveGenesisInt(record.Amount, "reward accumulator")
		if err != nil {
			panic(err)
		}
		k.SetRewardAccumulator(ctx, record.Denom, amount)
	}

	for _, record := range data.RewardPools {
		amount, err := types.ParsePositiveGenesisInt(record.Amount, "reward pool")
		if err != nil {
			panic(err)
		}
		k.SetRewardPoolAmount(ctx, record.Denom, amount)
	}

	for _, record := range data.RewardDebts {
		addr, err := sdk.AccAddressFromBech32(record.Address)
		if err != nil {
			panic(err)
		}
		amount, err := types.ParsePositiveGenesisInt(record.Amount, "reward debt")
		if err != nil {
			panic(err)
		}
		k.SetRewardDebt(ctx, addr, record.Denom, amount)
	}

	for _, record := range data.PendingRewards {
		addr, err := sdk.AccAddressFromBech32(record.Address)
		if err != nil {
			panic(err)
		}
		amount, err := types.ParsePositiveGenesisInt(record.Amount, "pending reward")
		if err != nil {
			panic(err)
		}
		k.SetPendingRewardAmount(ctx, addr, record.Denom, amount)
	}

	k.SetGovernanceEnabled(ctx, data.GovernanceEnabled)
}

// ExportGenesis exports x/dodxstaking state.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	gs := types.DefaultGenesisState()
	k.IterateStakes(ctx, func(addr sdk.AccAddress, amount sdkmath.Int) bool {
		gs.Stakes = append(gs.Stakes, types.StakeRecord{
			Address: addr.String(),
			Amount:  sdk.NewCoin(core.MicroDODxDenom, amount),
		})
		return false
	})

	k.IterateRewardDenoms(ctx, func(denom string) bool {
		acc := k.GetRewardAccumulator(ctx, denom)
		if acc.IsPositive() {
			gs.RewardAccumulators = append(gs.RewardAccumulators, types.RewardAmountRecord{
				Denom:  denom,
				Amount: acc.String(),
			})
		}

		pool := k.GetRewardPoolAmount(ctx, denom)
		if pool.IsPositive() {
			gs.RewardPools = append(gs.RewardPools, types.RewardAmountRecord{
				Denom:  denom,
				Amount: pool.String(),
			})
		}
		return false
	})

	k.IterateRewardDebts(ctx, func(addr sdk.AccAddress, denom string, amount sdkmath.Int) bool {
		if amount.IsPositive() {
			gs.RewardDebts = append(gs.RewardDebts, types.AccountRewardAmountRecord{
				Address: addr.String(),
				Denom:   denom,
				Amount:  amount.String(),
			})
		}
		return false
	})

	k.IteratePendingRewardAmounts(ctx, func(addr sdk.AccAddress, denom string, amount sdkmath.Int) bool {
		if amount.IsPositive() {
			gs.PendingRewards = append(gs.PendingRewards, types.AccountRewardAmountRecord{
				Address: addr.String(),
				Denom:   denom,
				Amount:  amount.String(),
			})
		}
		return false
	})

	gs.GovernanceEnabled = k.GovernanceEnabled(ctx)
	return gs
}
