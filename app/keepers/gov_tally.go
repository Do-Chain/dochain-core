package keepers

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	forktypes "github.com/Daviddochain/dochain-core/v4/types/fork"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	communityVoterPowerCapNumerator   int64 = 25
	communityVoterPowerCapDenominator int64 = 1000
)

func (appKeepers *AppKeepers) DoCommunityTallyFn() govkeeper.CalculateVoteResultsAndVotingPowerFn {
	return func(
		ctx context.Context,
		govKeeper govkeeper.Keeper,
		proposal v1.Proposal,
		validators map[string]v1.ValidatorGovInfo,
	) (sdkmath.LegacyDec, map[v1.VoteOption]sdkmath.LegacyDec, error) {
		if !doCommunityGovernanceActive(ctx) {
			return appKeepers.defaultCommunityTallyFn(ctx, govKeeper, proposal, validators)
		}

		if appKeepers.DODxStakingKeeper.GovernanceEnabled(sdk.UnwrapSDKContext(ctx)) {
			return appKeepers.dodxStakedTallyFn(ctx, govKeeper, proposal)
		}

		results := emptyCommunityTallyResults()

		totalBonded, err := appKeepers.StakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			return sdkmath.LegacyZeroDec(), nil, err
		}
		voterPowerCap := sdkmath.LegacyNewDecFromInt(totalBonded).
			MulInt64(communityVoterPowerCapNumerator).
			QuoInt64(communityVoterPowerCapDenominator)

		totalVotingPower := sdkmath.LegacyZeroDec()
		votesToRemove := []collections.Pair[uint64, sdk.AccAddress]{}
		rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)

		err = govKeeper.Votes.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], vote v1.Vote) (bool, error) {
			voter, err := appKeepers.AccountKeeper.AddressCodec().StringToBytes(vote.Voter)
			if err != nil {
				return false, err
			}

			valAddrStr, err := appKeepers.StakingKeeper.ValidatorAddressCodec().BytesToString(voter)
			if err != nil {
				return false, err
			}

			if _, ok := validators[valAddrStr]; ok {
				votesToRemove = append(votesToRemove, key)
				return false, nil
			}

			voterVotingPower := sdkmath.LegacyZeroDec()
			err = appKeepers.StakingKeeper.IterateDelegations(ctx, voter, func(_ int64, delegation stakingtypes.DelegationI) bool {
				valAddrStr := delegation.GetValidatorAddr()
				if val, ok := validators[valAddrStr]; ok {
					votingPower := delegation.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
					voterVotingPower = voterVotingPower.Add(votingPower)
				}

				return false
			})
			if err != nil {
				return false, err
			}

			votingPower := capCommunityVoterPower(voterVotingPower, voterPowerCap)
			for _, option := range vote.Options {
				weight, _ := sdkmath.LegacyNewDecFromStr(option.Weight)
				results[option.Option] = results[option.Option].Add(votingPower.Mul(weight))
			}
			totalVotingPower = totalVotingPower.Add(votingPower)

			votesToRemove = append(votesToRemove, key)
			return false, nil
		})
		if err != nil {
			return sdkmath.LegacyZeroDec(), nil, fmt.Errorf("error while iterating delegations: %w", err)
		}

		for _, key := range votesToRemove {
			if err := govKeeper.Votes.Remove(ctx, key); err != nil {
				return sdkmath.LegacyDec{}, nil, fmt.Errorf("error while removing vote (%d/%s): %w", key.K1(), key.K2(), err)
			}
		}

		return totalVotingPower, results, nil
	}
}

func (appKeepers *AppKeepers) dodxStakedTallyFn(
	ctx context.Context,
	govKeeper govkeeper.Keeper,
	proposal v1.Proposal,
) (sdkmath.LegacyDec, map[v1.VoteOption]sdkmath.LegacyDec, error) {
	totalVotingPower := sdkmath.LegacyZeroDec()
	results := emptyCommunityTallyResults()

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
	votesToRemove := []collections.Pair[uint64, sdk.AccAddress]{}

	err := govKeeper.Votes.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], vote v1.Vote) (bool, error) {
		voter, err := appKeepers.AccountKeeper.AddressCodec().StringToBytes(vote.Voter)
		if err != nil {
			return false, err
		}

		stakedDODx := appKeepers.DODxStakingKeeper.GetStakeAmount(sdkCtx, voter)
		votingPower := sdkmath.LegacyNewDecFromInt(stakedDODx)
		for _, option := range vote.Options {
			weight, _ := sdkmath.LegacyNewDecFromStr(option.Weight)
			results[option.Option] = results[option.Option].Add(votingPower.Mul(weight))
		}
		totalVotingPower = totalVotingPower.Add(votingPower)

		votesToRemove = append(votesToRemove, key)
		return false, nil
	})
	if err != nil {
		return sdkmath.LegacyZeroDec(), nil, fmt.Errorf("error while iterating DODx governance votes: %w", err)
	}

	for _, key := range votesToRemove {
		if err := govKeeper.Votes.Remove(ctx, key); err != nil {
			return sdkmath.LegacyDec{}, nil, fmt.Errorf("error while removing vote (%d/%s): %w", key.K1(), key.K2(), err)
		}
	}

	return totalVotingPower, results, nil
}

func (appKeepers *AppKeepers) defaultCommunityTallyFn(
	ctx context.Context,
	govKeeper govkeeper.Keeper,
	proposal v1.Proposal,
	validators map[string]v1.ValidatorGovInfo,
) (sdkmath.LegacyDec, map[v1.VoteOption]sdkmath.LegacyDec, error) {
	totalVotingPower := sdkmath.LegacyZeroDec()
	results := emptyCommunityTallyResults()

	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
	votesToRemove := []collections.Pair[uint64, sdk.AccAddress]{}
	err := govKeeper.Votes.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], vote v1.Vote) (bool, error) {
		voter, err := appKeepers.AccountKeeper.AddressCodec().StringToBytes(vote.Voter)
		if err != nil {
			return false, err
		}

		valAddrStr, err := appKeepers.StakingKeeper.ValidatorAddressCodec().BytesToString(voter)
		if err != nil {
			return false, err
		}
		if val, ok := validators[valAddrStr]; ok {
			val.Vote = vote.Options
			validators[valAddrStr] = val
		}

		err = appKeepers.StakingKeeper.IterateDelegations(ctx, voter, func(_ int64, delegation stakingtypes.DelegationI) bool {
			valAddrStr := delegation.GetValidatorAddr()

			if val, ok := validators[valAddrStr]; ok {
				val.DelegatorDeductions = val.DelegatorDeductions.Add(delegation.GetShares())
				validators[valAddrStr] = val

				votingPower := delegation.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
				for _, option := range vote.Options {
					weight, _ := sdkmath.LegacyNewDecFromStr(option.Weight)
					results[option.Option] = results[option.Option].Add(votingPower.Mul(weight))
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
			}

			return false
		})
		if err != nil {
			return false, err
		}

		votesToRemove = append(votesToRemove, key)
		return false, nil
	})
	if err != nil {
		return sdkmath.LegacyZeroDec(), nil, fmt.Errorf("error while iterating delegations: %w", err)
	}

	for _, key := range votesToRemove {
		if err := govKeeper.Votes.Remove(ctx, key); err != nil {
			return sdkmath.LegacyDec{}, nil, fmt.Errorf("error while removing vote (%d/%s): %w", key.K1(), key.K2(), err)
		}
	}

	for _, val := range validators {
		if len(val.Vote) == 0 {
			continue
		}

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)
		for _, option := range val.Vote {
			weight, _ := sdkmath.LegacyNewDecFromStr(option.Weight)
			results[option.Option] = results[option.Option].Add(votingPower.Mul(weight))
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}

	return totalVotingPower, results, nil
}

func emptyCommunityTallyResults() map[v1.VoteOption]sdkmath.LegacyDec {
	return map[v1.VoteOption]sdkmath.LegacyDec{
		v1.OptionYes:        sdkmath.LegacyZeroDec(),
		v1.OptionAbstain:    sdkmath.LegacyZeroDec(),
		v1.OptionNo:         sdkmath.LegacyZeroDec(),
		v1.OptionNoWithVeto: sdkmath.LegacyZeroDec(),
	}
}

func doCommunityGovernanceActive(ctx context.Context) bool {
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	return forktypes.DoCommunityGovernanceHeight > 0 && height >= forktypes.DoCommunityGovernanceHeight
}

func capCommunityVoterPower(votingPower, cap sdkmath.LegacyDec) sdkmath.LegacyDec {
	if cap.IsZero() || votingPower.LTE(cap) {
		return votingPower
	}

	return cap
}
