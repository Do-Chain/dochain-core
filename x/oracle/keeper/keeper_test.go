package keeper

import (
	"bytes"
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestExchangeRate(t *testing.T) {
	input := CreateTestInput(t)

	cnyDoExchangeRate := sdkmath.LegacyNewDecWithPrec(839, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)
	gbpDoExchangeRate := sdkmath.LegacyNewDecWithPrec(4995, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)
	krwDoExchangeRate := sdkmath.LegacyNewDecWithPrec(2838, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)
	doExchangeRate := sdkmath.LegacyNewDecWithPrec(3282384, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)

	// Set & get rates
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroCNYDenom, cnyDoExchangeRate)
	rate, err := input.OracleKeeper.GetDoExchangeRate(input.Ctx, core.MicroCNYDenom)
	require.NoError(t, err)
	require.Equal(t, cnyDoExchangeRate, rate)

	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroGBPDenom, gbpDoExchangeRate)
	rate, err = input.OracleKeeper.GetDoExchangeRate(input.Ctx, core.MicroGBPDenom)
	require.NoError(t, err)
	require.Equal(t, gbpDoExchangeRate, rate)

	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroKRWDenom, krwDoExchangeRate)
	rate, err = input.OracleKeeper.GetDoExchangeRate(input.Ctx, core.MicroKRWDenom)
	require.NoError(t, err)
	require.Equal(t, krwDoExchangeRate, rate)

	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroDoDenom, doExchangeRate)
	rate, _ = input.OracleKeeper.GetDoExchangeRate(input.Ctx, core.MicroDoDenom)
	require.Equal(t, sdkmath.LegacyOneDec(), rate)

	input.OracleKeeper.DeleteDoExchangeRate(input.Ctx, core.MicroKRWDenom)
	_, err = input.OracleKeeper.GetDoExchangeRate(input.Ctx, core.MicroKRWDenom)
	require.Error(t, err)

	numExchangeRates := 0
	handler := func(denom string, exchangeRate sdkmath.LegacyDec) (stop bool) {
		numExchangeRates++
		return false
	}
	input.OracleKeeper.IterateDoExchangeRates(input.Ctx, handler)

	require.True(t, numExchangeRates == 3)
}

func TestIterateDoExchangeRates(t *testing.T) {
	input := CreateTestInput(t)

	cnyDoExchangeRate := sdkmath.LegacyNewDecWithPrec(839, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)
	gbpDoExchangeRate := sdkmath.LegacyNewDecWithPrec(4995, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)
	krwDoExchangeRate := sdkmath.LegacyNewDecWithPrec(2838, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)
	doExchangeRate := sdkmath.LegacyNewDecWithPrec(3282384, int64(OracleDecPrecision)).MulInt64(core.MicroUnit)

	// Set & get rates
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroCNYDenom, cnyDoExchangeRate)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroGBPDenom, gbpDoExchangeRate)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroKRWDenom, krwDoExchangeRate)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroDoDenom, doExchangeRate)

	input.OracleKeeper.IterateDoExchangeRates(input.Ctx, func(denom string, rate sdkmath.LegacyDec) (stop bool) {
		switch denom {
		case core.MicroCNYDenom:
			require.Equal(t, cnyDoExchangeRate, rate)
		case core.MicroGBPDenom:
			require.Equal(t, gbpDoExchangeRate, rate)
		case core.MicroKRWDenom:
			require.Equal(t, krwDoExchangeRate, rate)
		case core.MicroDoDenom:
			require.Equal(t, doExchangeRate, rate)
		}
		return false
	})
}

func TestRewardPool(t *testing.T) {
	input := CreateTestInput(t)

	fees := sdk.NewCoins(sdk.NewCoin(core.MicroSDRDenom, sdkmath.NewInt(1000)))
	acc := input.AccountKeeper.GetModuleAccount(input.Ctx, types.ModuleName)
	err := FundAccount(input, acc.GetAddress(), fees)
	if err != nil {
		panic(err) // never occurs
	}

	KFees := input.OracleKeeper.GetRewardPool(input.Ctx, core.MicroSDRDenom)
	require.Equal(t, fees[0], KFees)
}

func TestParams(t *testing.T) {
	input := CreateTestInput(t)

	// Test default params setting
	input.OracleKeeper.SetParams(input.Ctx, types.DefaultParams())
	params := input.OracleKeeper.GetParams(input.Ctx)
	require.NotNil(t, params)

	// Test custom params setting
	votePeriod := uint64(10)
	voteThreshold := sdkmath.LegacyNewDecWithPrec(33, 2)
	oracleRewardBand := sdkmath.LegacyNewDecWithPrec(1, 2)
	rewardDistributionWindow := uint64(10000000000000)
	slashFraction := sdkmath.LegacyNewDecWithPrec(1, 2)
	slashWindow := uint64(1000)
	minValidPerWindow := sdkmath.LegacyNewDecWithPrec(1, 4)
	whitelist := types.DenomList{
		{Name: core.MicroSDRDenom, TobinTax: types.DefaultTobinTax},
		{Name: core.MicroKRWDenom, TobinTax: types.DefaultTobinTax},
	}

	// Should really test validateParams, but skipping because obvious
	newParams := types.Params{
		VotePeriod:               votePeriod,
		VoteThreshold:            voteThreshold,
		RewardBand:               oracleRewardBand,
		RewardDistributionWindow: rewardDistributionWindow,
		Whitelist:                whitelist,
		SlashFraction:            slashFraction,
		SlashWindow:              slashWindow,
		MinValidPerWindow:        minValidPerWindow,
	}
	input.OracleKeeper.SetParams(input.Ctx, newParams)

	storedParams := input.OracleKeeper.GetParams(input.Ctx)
	require.NotNil(t, storedParams)
	require.Equal(t, storedParams, newParams)
}

func TestFeederDelegation(t *testing.T) {
	input := CreateTestInput(t)

	// Test default getters and setters
	delegate := input.OracleKeeper.GetFeederDelegation(input.Ctx, ValAddrs[0])
	require.Equal(t, Addrs[0], delegate)

	input.OracleKeeper.SetFeederDelegation(input.Ctx, ValAddrs[0], Addrs[1])
	delegate = input.OracleKeeper.GetFeederDelegation(input.Ctx, ValAddrs[0])
	require.Equal(t, Addrs[1], delegate)
}

func TestIterateFeederDelegations(t *testing.T) {
	input := CreateTestInput(t)

	// Test default getters and setters
	delegate := input.OracleKeeper.GetFeederDelegation(input.Ctx, ValAddrs[0])
	require.Equal(t, Addrs[0], delegate)

	input.OracleKeeper.SetFeederDelegation(input.Ctx, ValAddrs[0], Addrs[1])

	var delegators []sdk.ValAddress
	var delegates []sdk.AccAddress
	input.OracleKeeper.IterateFeederDelegations(input.Ctx, func(delegator sdk.ValAddress, delegate sdk.AccAddress) (stop bool) {
		delegators = append(delegators, delegator)
		delegates = append(delegates, delegate)
		return false
	})

	require.Equal(t, 1, len(delegators))
	require.Equal(t, 1, len(delegates))
	require.Equal(t, ValAddrs[0], delegators[0])
	require.Equal(t, Addrs[1], delegates[0])
}

func TestMissCounter(t *testing.T) {
	input := CreateTestInput(t)

	// Test default getters and setters
	counter := input.OracleKeeper.GetMissCounter(input.Ctx, ValAddrs[0])
	require.Equal(t, uint64(0), counter)

	missCounter := uint64(10)
	input.OracleKeeper.SetMissCounter(input.Ctx, ValAddrs[0], missCounter)
	counter = input.OracleKeeper.GetMissCounter(input.Ctx, ValAddrs[0])
	require.Equal(t, missCounter, counter)

	input.OracleKeeper.DeleteMissCounter(input.Ctx, ValAddrs[0])
	counter = input.OracleKeeper.GetMissCounter(input.Ctx, ValAddrs[0])
	require.Equal(t, uint64(0), counter)
}

func TestIterateMissCounters(t *testing.T) {
	input := CreateTestInput(t)

	// Test default getters and setters
	counter := input.OracleKeeper.GetMissCounter(input.Ctx, ValAddrs[0])
	require.Equal(t, uint64(0), counter)

	missCounter := uint64(10)
	input.OracleKeeper.SetMissCounter(input.Ctx, ValAddrs[1], missCounter)

	var operators []sdk.ValAddress
	var missCounters []uint64
	input.OracleKeeper.IterateMissCounters(input.Ctx, func(delegator sdk.ValAddress, missCounter uint64) (stop bool) {
		operators = append(operators, delegator)
		missCounters = append(missCounters, missCounter)
		return false
	})

	require.Equal(t, 1, len(operators))
	require.Equal(t, 1, len(missCounters))
	require.Equal(t, ValAddrs[1], operators[0])
	require.Equal(t, missCounter, missCounters[0])
}

func TestAggregatePrevoteAddDelete(t *testing.T) {
	input := CreateTestInput(t)

	hash := types.GetAggregateVoteHash("salt", "100ukrw,1000uusd", sdk.ValAddress(Addrs[0]))
	aggregatePrevote := types.NewAggregateDoRatePrevote(hash, sdk.ValAddress(Addrs[0]), 0)
	input.OracleKeeper.SetAggregateDoRatePrevote(input.Ctx, sdk.ValAddress(Addrs[0]), aggregatePrevote)

	KPrevote, err := input.OracleKeeper.GetAggregateDoRatePrevote(input.Ctx, sdk.ValAddress(Addrs[0]))
	require.NoError(t, err)
	require.Equal(t, aggregatePrevote, KPrevote)

	input.OracleKeeper.DeleteAggregateDoRatePrevote(input.Ctx, sdk.ValAddress(Addrs[0]))
	_, err = input.OracleKeeper.GetAggregateDoRatePrevote(input.Ctx, sdk.ValAddress(Addrs[0]))
	require.Error(t, err)
}

func TestAggregatePrevoteIterate(t *testing.T) {
	input := CreateTestInput(t)

	hash := types.GetAggregateVoteHash("salt", "100ukrw,1000uusd", sdk.ValAddress(Addrs[0]))
	aggregatePrevote1 := types.NewAggregateDoRatePrevote(hash, sdk.ValAddress(Addrs[0]), 0)
	input.OracleKeeper.SetAggregateDoRatePrevote(input.Ctx, sdk.ValAddress(Addrs[0]), aggregatePrevote1)

	hash2 := types.GetAggregateVoteHash("salt", "100ukrw,1000uusd", sdk.ValAddress(Addrs[1]))
	aggregatePrevote2 := types.NewAggregateDoRatePrevote(hash2, sdk.ValAddress(Addrs[1]), 0)
	input.OracleKeeper.SetAggregateDoRatePrevote(input.Ctx, sdk.ValAddress(Addrs[1]), aggregatePrevote2)

	i := 0
	bigger := bytes.Compare(Addrs[0], Addrs[1])
	input.OracleKeeper.IterateAggregateDoRatePrevotes(input.Ctx, func(voter sdk.ValAddress, p types.AggregateDoRatePrevote) (stop bool) {
		if (i == 0 && bigger == -1) || (i == 1 && bigger == 1) {
			require.Equal(t, aggregatePrevote1, p)
			require.Equal(t, voter.String(), p.Voter)
		} else {
			require.Equal(t, aggregatePrevote2, p)
			require.Equal(t, voter.String(), p.Voter)
		}

		i++
		return false
	})
}

func TestAggregateVoteAddDelete(t *testing.T) {
	input := CreateTestInput(t)

	aggregateVote := types.NewAggregateDoRateVote(types.DoRateTuples{
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(-1)},
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(0)},
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(1)},
	}, sdk.ValAddress(Addrs[0]))
	input.OracleKeeper.SetAggregateDoRateVote(input.Ctx, sdk.ValAddress(Addrs[0]), aggregateVote)

	KVote, err := input.OracleKeeper.GetAggregateDoRateVote(input.Ctx, sdk.ValAddress(Addrs[0]))
	require.NoError(t, err)
	require.Equal(t, aggregateVote, KVote)

	input.OracleKeeper.DeleteAggregateDoRateVote(input.Ctx, sdk.ValAddress(Addrs[0]))
	_, err = input.OracleKeeper.GetAggregateDoRateVote(input.Ctx, sdk.ValAddress(Addrs[0]))
	require.Error(t, err)
}

func TestAggregateVoteIterate(t *testing.T) {
	input := CreateTestInput(t)

	aggregateVote1 := types.NewAggregateDoRateVote(types.DoRateTuples{
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(-1)},
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(0)},
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(1)},
	}, sdk.ValAddress(Addrs[0]))
	input.OracleKeeper.SetAggregateDoRateVote(input.Ctx, sdk.ValAddress(Addrs[0]), aggregateVote1)

	aggregateVote2 := types.NewAggregateDoRateVote(types.DoRateTuples{
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(-1)},
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(0)},
		{Denom: "foo", ExchangeRate: sdkmath.LegacyNewDec(1)},
	}, sdk.ValAddress(Addrs[1]))
	input.OracleKeeper.SetAggregateDoRateVote(input.Ctx, sdk.ValAddress(Addrs[1]), aggregateVote2)

	i := 0
	bigger := bytes.Compare(address.MustLengthPrefix(Addrs[0]), address.MustLengthPrefix(Addrs[1]))
	input.OracleKeeper.IterateAggregateDoRateVotes(input.Ctx, func(voter sdk.ValAddress, p types.AggregateDoRateVote) (stop bool) {
		if (i == 0 && bigger == -1) || (i == 1 && bigger == 1) {
			require.Equal(t, aggregateVote1, p)
			require.Equal(t, voter.String(), p.Voter)
		} else {
			require.Equal(t, aggregateVote2, p)
			require.Equal(t, voter.String(), p.Voter)
		}

		i++
		return false
	})
}

func TestTobinTaxGetSet(t *testing.T) {
	input := CreateTestInput(t)
	input.OracleKeeper.ClearTobinTaxes(input.Ctx)

	tobinTaxes := map[string]sdkmath.LegacyDec{
		core.MicroSDRDenom: sdkmath.LegacyNewDec(1),
		core.MicroUSDDenom: sdkmath.LegacyNewDecWithPrec(1, 3),
		core.MicroKRWDenom: sdkmath.LegacyNewDecWithPrec(123, 3),
		core.MicroMNTDenom: sdkmath.LegacyNewDecWithPrec(1423, 4),
	}

	for denom, tobinTax := range tobinTaxes {
		input.OracleKeeper.SetTobinTax(input.Ctx, denom, tobinTax)
		factor, err := input.OracleKeeper.GetTobinTax(input.Ctx, denom)
		require.NoError(t, err)
		require.Equal(t, tobinTaxes[denom], factor)
	}

	input.OracleKeeper.IterateTobinTaxes(input.Ctx, func(denom string, tobinTax sdkmath.LegacyDec) (stop bool) {
		require.Equal(t, tobinTaxes[denom], tobinTax)
		return false
	})

	input.OracleKeeper.ClearTobinTaxes(input.Ctx)
	for denom := range tobinTaxes {
		_, err := input.OracleKeeper.GetTobinTax(input.Ctx, denom)
		require.Error(t, err)
	}
}

func TestValidateFeeder(t *testing.T) {
	// initial setup
	input := CreateTestInput(t)
	addr, val := ValAddrs[0], ValPubKeys[0]
	addr1, val1 := ValAddrs[1], ValPubKeys[1]
	amt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	ctx := input.Ctx

	// Validator created
	_, err := stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(addr, val, amt))
	require.NoError(t, err)
	_, err = stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(addr1, val1, amt))
	require.NoError(t, err)
	input.StakingKeeper.EndBlocker(ctx)

	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr)),
		sdk.NewCoins(sdk.NewCoin(params.BondDenom, InitTokens.Sub(amt))),
	)
	validator, err := input.StakingKeeper.Validator(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, amt, validator.GetBondedTokens())
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr1)),
		sdk.NewCoins(sdk.NewCoin(params.BondDenom, InitTokens.Sub(amt))),
	)
	validator, err = input.StakingKeeper.Validator(ctx, addr1)
	require.NoError(t, err)
	require.Equal(t, amt, validator.GetBondedTokens())

	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr), addr))
	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr1))

	// delegate works
	input.OracleKeeper.SetFeederDelegation(input.Ctx, addr, sdk.AccAddress(addr1))
	require.NoError(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr))
	require.Error(t, input.OracleKeeper.ValidateFeeder(input.Ctx, Addrs[2], addr))

	// only active validators can do oracle votes
	validator, err = input.StakingKeeper.GetValidator(input.Ctx, addr)
	require.NoError(t, err)
	sValidator := validator.(stakingtypes.Validator)
	sValidator.Status = stakingtypes.Unbonded
	input.StakingKeeper.SetValidator(input.Ctx, sValidator)
	require.Error(t, input.OracleKeeper.ValidateFeeder(input.Ctx, sdk.AccAddress(addr1), addr))
}
