package oracle_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/x/oracle"
	"github.com/Daviddochain/dochain-core/v4/x/oracle/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	"github.com/stretchr/testify/require"
)

func TestExportInitGenesis(t *testing.T) {
	input, _ := setup(t)

	input.OracleKeeper.SetFeederDelegation(input.Ctx, keeper.ValAddrs[0], keeper.Addrs[1])
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, "denom", sdkmath.LegacyNewDec(123))
	input.OracleKeeper.SetAggregateDoRatePrevote(input.Ctx, keeper.ValAddrs[0], types.NewAggregateDoRatePrevote(types.AggregateVoteHash{123}, keeper.ValAddrs[0], uint64(2)))
	input.OracleKeeper.SetAggregateDoRateVote(input.Ctx, keeper.ValAddrs[0], types.NewAggregateDoRateVote(types.DoRateTuples{{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(123)}}, keeper.ValAddrs[0]))
	input.OracleKeeper.SetTobinTax(input.Ctx, "denom", sdkmath.LegacyNewDecWithPrec(123, 3))
	input.OracleKeeper.SetTobinTax(input.Ctx, "denom2", sdkmath.LegacyNewDecWithPrec(123, 3))
	input.OracleKeeper.SetMissCounter(input.Ctx, keeper.ValAddrs[0], 10)
	genesis := oracle.ExportGenesis(input.Ctx, input.OracleKeeper)

	newInput := keeper.CreateTestInput(t)
	oracle.InitGenesis(newInput.Ctx, newInput.OracleKeeper, genesis)
	newGenesis := oracle.ExportGenesis(newInput.Ctx, newInput.OracleKeeper)

	require.Equal(t, genesis, newGenesis)
}

func TestInitGenesis(t *testing.T) {
	input, _ := setup(t)
	genesis := types.DefaultGenesisState()
	require.NotPanics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})

	genesis.FeederDelegations = []types.FeederDelegation{{
		FeederAddress:    keeper.Addrs[0].String(),
		ValidatorAddress: "invalid",
	}}

	require.Panics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})

	genesis.FeederDelegations = []types.FeederDelegation{{
		FeederAddress:    "invalid",
		ValidatorAddress: keeper.ValAddrs[0].String(),
	}}

	require.Panics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})

	genesis.FeederDelegations = []types.FeederDelegation{{
		FeederAddress:    keeper.Addrs[0].String(),
		ValidatorAddress: keeper.ValAddrs[0].String(),
	}}

	genesis.MissCounters = []types.MissCounter{
		{
			ValidatorAddress: "invalid",
			MissCounter:      10,
		},
	}

	require.Panics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})

	genesis.MissCounters = []types.MissCounter{
		{
			ValidatorAddress: keeper.ValAddrs[0].String(),
			MissCounter:      10,
		},
	}

	genesis.AggregateExchangeRatePrevotes = []types.AggregateDoRatePrevote{
		{
			Hash:        "hash",
			Voter:       "invalid",
			SubmitBlock: 100,
		},
	}

	require.Panics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})

	genesis.AggregateExchangeRatePrevotes = []types.AggregateDoRatePrevote{
		{
			Hash:        "hash",
			Voter:       keeper.ValAddrs[0].String(),
			SubmitBlock: 100,
		},
	}

	genesis.AggregateExchangeRateVotes = []types.AggregateDoRateVote{
		{
			ExchangeRateTuples: []types.DoRateTuple{
				{
					Denom:        "ukrw",
					ExchangeRate: sdkmath.LegacyNewDec(10),
				},
			},
			Voter: "invalid",
		},
	}

	require.Panics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})

	genesis.AggregateExchangeRateVotes = []types.AggregateDoRateVote{
		{
			ExchangeRateTuples: []types.DoRateTuple{
				{
					Denom:        "ukrw",
					ExchangeRate: sdkmath.LegacyNewDec(10),
				},
			},
			Voter: keeper.ValAddrs[0].String(),
		},
	}

	require.NotPanics(t, func() {
		oracle.InitGenesis(input.Ctx, input.OracleKeeper, genesis)
	})
}
