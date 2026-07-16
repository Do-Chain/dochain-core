package keeper

import (
	"cosmossdk.io/math"
	forktypes "github.com/Daviddochain/dochain-core/v4/types/fork"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SlashAndResetMissCounters penalizes operators who miss too many oracle votes and clears miss counters.
func (k Keeper) SlashAndResetMissCounters(ctx sdk.Context) {
	height := ctx.BlockHeight()
	distributionHeight := height - sdk.ValidatorUpdateDelay - 1

	// slash_window / vote_period
	votePeriodsPerWindow := uint64(
		math.LegacyNewDec(int64(k.SlashWindow(ctx))).
			QuoInt64(int64(k.VotePeriod(ctx))).
			TruncateInt64(),
	)
	minValidPerWindow := k.MinValidPerWindow(ctx)
	slashFraction := k.SlashFraction(ctx)
	powerReduction := k.StakingKeeper.PowerReduction(ctx)

	k.IterateMissCounters(ctx, func(operator sdk.ValAddress, missCounter uint64) bool {
		// Calculate valid vote rate; (SlashWindow - MissCounter)/SlashWindow
		validVoteRate := math.LegacyNewDecFromInt(
			math.NewInt(int64(votePeriodsPerWindow - missCounter))).
			QuoInt64(int64(votePeriodsPerWindow))

		// Penalize the validator whose the valid vote rate is smaller than min threshold
		if validVoteRate.LT(minValidPerWindow) {
			validator, err := k.StakingKeeper.Validator(ctx, operator)
			if err != nil {
				return false
			}
			if validator.IsBonded() && !validator.IsJailed() {
				consAddr, err := validator.GetConsAddr()
				if err != nil {
					panic(err)
				}

				if !doOracleJailOnlyActive(ctx) {
					k.StakingKeeper.Slash(
						ctx, consAddr,
						distributionHeight, validator.GetConsensusPower(powerReduction), slashFraction,
					)
				}
				k.StakingKeeper.Jail(ctx, consAddr)
			}
		}

		k.DeleteMissCounter(ctx, operator)
		return false
	})
}

func doOracleJailOnlyActive(ctx sdk.Context) bool {
	return forktypes.DoCommunityGovernanceHeight > 0 && ctx.BlockHeight() >= forktypes.DoCommunityGovernanceHeight
}
