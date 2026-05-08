package keeper

import (
	"sort"
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
)

func TestOrganizeAggregate(t *testing.T) {
	input := CreateTestInput(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	input.StakingKeeper.EndBlocker(ctx)

	sdrBallot := types.ExchangeRateBallot{
		types.NewVoteForTally(sdkmath.LegacyNewDec(17), core.MicroSDRDenom, ValAddrs[0], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(10), core.MicroSDRDenom, ValAddrs[1], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(6), core.MicroSDRDenom, ValAddrs[2], power),
	}
	krwBallot := types.ExchangeRateBallot{
		types.NewVoteForTally(sdkmath.LegacyNewDec(1000), core.MicroKRWDenom, ValAddrs[0], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(1300), core.MicroKRWDenom, ValAddrs[1], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(2000), core.MicroKRWDenom, ValAddrs[2], power),
	}

	for i := range sdrBallot {
		input.OracleKeeper.SetAggregateDoRateVote(input.Ctx, ValAddrs[i],
			types.NewAggregateDoRateVote(types.DoRateTuples{
				{Denom: sdrBallot[i].Denom, ExchangeRate: sdrBallot[i].ExchangeRate},
				{Denom: krwBallot[i].Denom, ExchangeRate: krwBallot[i].ExchangeRate},
			}, ValAddrs[i]))
	}

	// organize votes by denom
	ballotMap := input.OracleKeeper.OrganizeBallotByDenom(input.Ctx, map[string]types.Claim{
		ValAddrs[0].String(): {
			Power:     power,
			WinCount:  0,
			Recipient: ValAddrs[0],
		},
		ValAddrs[1].String(): {
			Power:     power,
			WinCount:  0,
			Recipient: ValAddrs[1],
		},
		ValAddrs[2].String(): {
			Power:     power,
			WinCount:  0,
			Recipient: ValAddrs[2],
		},
	})

	// sort each ballot for comparison
	sort.Sort(sdrBallot)
	sort.Sort(krwBallot)
	sort.Sort(ballotMap[core.MicroSDRDenom])
	sort.Sort(ballotMap[core.MicroKRWDenom])

	require.Equal(t, sdrBallot, ballotMap[core.MicroSDRDenom])
	require.Equal(t, krwBallot, ballotMap[core.MicroKRWDenom])
}

func TestClearBallots(t *testing.T) {
	input := CreateTestInput(t)

	power := int64(100)
	amt := sdk.TokensFromConsensusPower(power, sdk.DefaultPowerReduction)
	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(ValAddrs[0], ValPubKeys[0], amt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(ValAddrs[1], ValPubKeys[1], amt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(ValAddrs[2], ValPubKeys[2], amt))
	require.NoError(t, err)
	input.StakingKeeper.EndBlocker(ctx)

	sdrBallot := types.ExchangeRateBallot{
		types.NewVoteForTally(sdkmath.LegacyNewDec(17), core.MicroSDRDenom, ValAddrs[0], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(10), core.MicroSDRDenom, ValAddrs[1], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(6), core.MicroSDRDenom, ValAddrs[2], power),
	}
	krwBallot := types.ExchangeRateBallot{
		types.NewVoteForTally(sdkmath.LegacyNewDec(1000), core.MicroKRWDenom, ValAddrs[0], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(1300), core.MicroKRWDenom, ValAddrs[1], power),
		types.NewVoteForTally(sdkmath.LegacyNewDec(2000), core.MicroKRWDenom, ValAddrs[2], power),
	}

	for i := range sdrBallot {
		input.OracleKeeper.SetAggregateDoRatePrevote(input.Ctx, ValAddrs[i], types.AggregateDoRatePrevote{
			Hash:        "",
			Voter:       ValAddrs[i].String(),
			SubmitBlock: uint64(input.Ctx.BlockHeight()),
		})

		input.OracleKeeper.SetAggregateDoRateVote(input.Ctx, ValAddrs[i],
			types.NewAggregateDoRateVote(types.DoRateTuples{
				{Denom: sdrBallot[i].Denom, ExchangeRate: sdrBallot[i].ExchangeRate},
				{Denom: krwBallot[i].Denom, ExchangeRate: krwBallot[i].ExchangeRate},
			}, ValAddrs[i]))
	}

	input.OracleKeeper.ClearBallots(input.Ctx, 5)

	prevoteCounter := 0
	voteCounter := 0
	input.OracleKeeper.IterateAggregateDoRatePrevotes(input.Ctx, func(_ sdk.ValAddress, _ types.AggregateDoRatePrevote) bool {
		prevoteCounter++
		return false
	})
	input.OracleKeeper.IterateAggregateDoRateVotes(input.Ctx, func(_ sdk.ValAddress, _ types.AggregateDoRateVote) bool {
		voteCounter++
		return false
	})

	require.Equal(t, prevoteCounter, 3)
	require.Equal(t, voteCounter, 0)

	input.OracleKeeper.ClearBallots(input.Ctx.WithBlockHeight(input.Ctx.BlockHeight()+6), 5)

	prevoteCounter = 0
	input.OracleKeeper.IterateAggregateDoRatePrevotes(input.Ctx, func(_ sdk.ValAddress, _ types.AggregateDoRatePrevote) bool {
		prevoteCounter++
		return false
	})
	require.Equal(t, prevoteCounter, 0)
}

func TestApplyWhitelist(t *testing.T) {
	input := CreateTestInput(t)

	// no update
	input.OracleKeeper.ApplyWhitelist(input.Ctx, types.DenomList{
		{
			Name:     core.MicroUSDDenom,
			TobinTax: sdkmath.LegacyOneDec(),
		},
		{
			Name:     "ukrw",
			TobinTax: sdkmath.LegacyOneDec(),
		},
	}, map[string]sdkmath.LegacyDec{
		core.MicroUSDDenom: sdkmath.LegacyZeroDec(),
		core.MicroKRWDenom: sdkmath.LegacyZeroDec(),
	})

	price, err := input.OracleKeeper.GetTobinTax(input.Ctx, core.MicroUSDDenom)
	require.NoError(t, err)
	require.Equal(t, price, sdkmath.LegacyOneDec())

	price, err = input.OracleKeeper.GetTobinTax(input.Ctx, "ukrw")
	require.NoError(t, err)
	require.Equal(t, price, sdkmath.LegacyOneDec())

	metadata, ok := input.BankKeeper.GetDenomMetaData(input.Ctx, core.MicroUSDDenom)
	require.True(t, ok)
	require.Equal(t, metadata.Base, core.MicroUSDDenom)
	require.Equal(t, metadata.Display, "usd")
	require.Equal(t, len(metadata.DenomUnits), 3)
	require.Equal(t, metadata.Description, "The native stable token of the do Columbus.")

	metadata, ok = input.BankKeeper.GetDenomMetaData(input.Ctx, "ukrw")
	require.True(t, ok)
	require.Equal(t, metadata.Base, "ukrw")
	require.Equal(t, metadata.Display, "krw")
	require.Equal(t, len(metadata.DenomUnits), 3)
	require.Equal(t, metadata.Description, "The native stable token of the do Columbus.")
}
