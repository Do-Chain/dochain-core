package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestSlashAndResetMissCounters(t *testing.T) {
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
	v0, err := input.StakingKeeper.Validator(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, amt, v0.GetBondedTokens())
	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(addr1)),
		sdk.NewCoins(sdk.NewCoin(params.BondDenom, InitTokens.Sub(amt))),
	)
	v1, err := input.StakingKeeper.Validator(ctx, addr1)
	require.NoError(t, err)
	require.Equal(t, amt, v1.GetBondedTokens())

	votePeriodsPerWindow := int64(input.OracleKeeper.SlashWindow(input.Ctx)) / int64(input.OracleKeeper.VotePeriod(input.Ctx))
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	minValidVotes := input.OracleKeeper.MinValidPerWindow(input.Ctx).MulInt64(votePeriodsPerWindow).TruncateInt64()
	// Case 1, no slash
	input.OracleKeeper.SetMissCounter(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	input.StakingKeeper.EndBlocker(ctx)

	validator, err := input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.NoError(t, err)
	require.Equal(t, amt, validator.GetBondedTokens())

	// Case 2, slash
	input.OracleKeeper.SetMissCounter(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, err = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.NoError(t, err)
	require.Equal(t, amt.Sub(slashFraction.MulInt(amt).TruncateInt()), validator.GetBondedTokens())
	require.True(t, validator.IsJailed())

	// Case 3, slash unbonded validator
	validator, err = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.NoError(t, err)
	validator.Status = stakingtypes.Unbonded
	validator.Jailed = false
	validator.Tokens = amt
	input.StakingKeeper.SetValidator(input.Ctx, validator)

	input.OracleKeeper.SetMissCounter(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, err = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.NoError(t, err)
	require.Equal(t, amt, validator.Tokens)
	require.False(t, validator.IsJailed())

	// Case 4, slash jailed validator
	validator, err = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.NoError(t, err)
	validator.Status = stakingtypes.Bonded
	validator.Jailed = true
	validator.Tokens = amt
	input.StakingKeeper.SetValidator(input.Ctx, validator)

	input.OracleKeeper.SetMissCounter(input.Ctx, ValAddrs[0], uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)
	validator, err = input.StakingKeeper.GetValidator(input.Ctx, ValAddrs[0])
	require.NoError(t, err)
	require.Equal(t, amt, validator.Tokens)
}

func TestOracleSlashOnlyBurnsValidatorSelfDelegation(t *testing.T) {
	input := CreateTestInput(t)
	operator, pubKey := ValAddrs[0], ValPubKeys[0]
	selfAmt := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	externalAmt := sdk.TokensFromConsensusPower(50, sdk.DefaultPowerReduction)
	stakingMsgSvr := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	ctx := input.Ctx

	_, err := stakingMsgSvr.CreateValidator(ctx, NewTestMsgCreateValidator(operator, pubKey, selfAmt))
	require.NoError(t, err)
	input.StakingKeeper.EndBlocker(ctx)

	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	_, err = stakingMsgSvr.Delegate(ctx, stakingtypes.NewMsgDelegate(Addrs[1].String(), operator.String(), sdk.NewCoin(params.BondDenom, externalAmt)))
	require.NoError(t, err)

	votePeriodsPerWindow := int64(input.OracleKeeper.SlashWindow(input.Ctx)) / int64(input.OracleKeeper.VotePeriod(input.Ctx))
	minValidVotes := input.OracleKeeper.MinValidPerWindow(input.Ctx).MulInt64(votePeriodsPerWindow).TruncateInt64()
	slashFraction := input.OracleKeeper.SlashFraction(input.Ctx)
	selfSlash := slashFraction.MulInt(selfAmt).TruncateInt()

	input.OracleKeeper.SetMissCounter(input.Ctx, operator, uint64(votePeriodsPerWindow-minValidVotes+1))
	input.OracleKeeper.SlashAndResetMissCounters(input.Ctx)

	validator, err := input.StakingKeeper.GetValidator(input.Ctx, operator)
	require.NoError(t, err)
	require.Equal(t, selfAmt.Add(externalAmt).Sub(selfSlash), validator.GetBondedTokens())
	require.True(t, validator.IsJailed())

	selfDelegation, err := input.StakingKeeper.GetDelegation(input.Ctx, sdk.AccAddress(operator), operator)
	require.NoError(t, err)
	require.Equal(t, selfAmt.Sub(selfSlash), validator.TokensFromShares(selfDelegation.Shares).TruncateInt())

	externalDelegation, err := input.StakingKeeper.GetDelegation(input.Ctx, Addrs[1], operator)
	require.NoError(t, err)
	require.Equal(t, externalAmt, validator.TokensFromShares(externalDelegation.Shares).TruncateInt())
}
