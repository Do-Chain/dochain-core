package keeper

import (
	"errors"

	"cosmossdk.io/math"
	forktypes "github.com/Daviddochain/dochain-core/v4/types/fork"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// SlashAndResetMissCounters penalizes operators who miss too many oracle votes and clears miss counters.
func (k Keeper) SlashAndResetMissCounters(ctx sdk.Context) {
	// slash_window / vote_period
	votePeriodsPerWindow := uint64(
		math.LegacyNewDec(int64(k.SlashWindow(ctx))).
			QuoInt64(int64(k.VotePeriod(ctx))).
			TruncateInt64(),
	)
	minValidPerWindow := k.MinValidPerWindow(ctx)
	slashFraction := k.SlashFraction(ctx)

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
					if _, err := k.slashValidatorSelfDelegation(ctx, operator, slashFraction); err != nil {
						ctx.Logger().Error("failed to slash validator self delegation", "validator", operator.String(), "error", err)
					}
				}
				k.StakingKeeper.Jail(ctx, consAddr)
			}
		}

		k.DeleteMissCounter(ctx, operator)
		return false
	})
}

func (k Keeper) slashValidatorSelfDelegation(ctx sdk.Context, operator sdk.ValAddress, slashFraction math.LegacyDec) (math.Int, error) {
	if slashFraction.IsZero() {
		return math.ZeroInt(), nil
	}

	validator, err := k.StakingKeeper.Validator(ctx, operator)
	if err != nil {
		return math.ZeroInt(), err
	}

	selfDelegator := sdk.AccAddress(operator)
	delegation, err := k.StakingKeeper.GetDelegation(ctx, selfDelegator, operator)
	if errors.Is(err, stakingtypes.ErrNoDelegation) {
		return math.ZeroInt(), nil
	}
	if err != nil {
		return math.ZeroInt(), err
	}

	selfDelegationTokens := validator.TokensFromShares(delegation.Shares).TruncateInt()
	tokensToSlash := slashFraction.MulInt(selfDelegationTokens).TruncateInt()
	if tokensToSlash.IsZero() {
		return math.ZeroInt(), nil
	}

	sharesToSlash := delegation.Shares.Mul(slashFraction)
	if sharesToSlash.IsZero() {
		return math.ZeroInt(), nil
	}
	if sharesToSlash.GT(delegation.Shares) {
		sharesToSlash = delegation.Shares
	}

	slashedTokens, err := k.StakingKeeper.Unbond(ctx, selfDelegator, operator, sharesToSlash)
	if err != nil {
		return math.ZeroInt(), err
	}
	if slashedTokens.IsZero() {
		return math.ZeroInt(), nil
	}

	bondDenom, err := k.StakingKeeper.BondDenom(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}
	if err := k.bankKeeper.BurnCoins(ctx, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(bondDenom, slashedTokens))); err != nil {
		return math.ZeroInt(), err
	}
	return slashedTokens, nil
}

func doOracleJailOnlyActive(ctx sdk.Context) bool {
	return forktypes.DoCommunityGovernanceHeight > 0 && ctx.BlockHeight() >= forktypes.DoCommunityGovernanceHeight
}
