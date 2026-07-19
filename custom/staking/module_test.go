package staking_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	apptesting "github.com/Daviddochain/dochain-core/v4/app/testing"
	customstaking "github.com/Daviddochain/dochain-core/v4/custom/staking"
	"github.com/Daviddochain/dochain-core/v4/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
)

type StakingTestSuite struct {
	apptesting.KeeperTestHelper
}

func TestStakingTestSuite(t *testing.T) {
	if !apptesting.WasmVMAvailable {
		t.Skip("app integration tests require a CGO-enabled WasmVM build")
	}
	suite.Run(t, new(StakingTestSuite))
}

// go test -v -run=TestStakingTestSuite/TestValidatorVPLimit github.com/Daviddochain/dochain-core/v4/custom/staking
func (s *StakingTestSuite) TestValidatorVPLimit() {
	s.Setup(s.T(), types.ColumbusChainID)

	// construct new validators, to a total of 10 validators, each with 10% of the total voting power
	num := 9
	addrDels := s.RandomAccountAddresses(num)
	for i, addrDel := range addrDels {
		s.FundAcc(addrDel, sdk.NewCoins(sdk.NewInt64Coin("udo", 1000000)))
		err := s.App.BankKeeper.DelegateCoinsFromAccountToModule(s.Ctx, addrDels[i], stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewInt64Coin("udo", 1000000)))
		s.Require().NoError(err)
	}
	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrDels)
	PKs := simtestutil.CreateTestPubKeys(num)

	var amts [9]math.Int
	for i := range amts {
		amts[i] = math.NewInt(1000000)
	}

	var validators [9]stakingtypes.Validator
	for i, amt := range amts {
		validators[i] = testutil.NewValidator(s.T(), valAddrs[i], PKs[i])
		validators[i], _ = validators[i].AddTokensFromDel(amt)
	}

	for i := range validators {
		validators[i] = stakingkeeper.TestingUpdateValidator(s.App.StakingKeeper, s.Ctx, validators[i], true)
	}

	// delegate to a validator over 20% VP
	s.FundAcc(s.TestAccs[0], sdk.NewCoins(sdk.NewInt64Coin("udo", 2000000)))
	s.App.DistrKeeper.SetValidatorHistoricalRewards(s.Ctx, valAddrs[0], 1, disttypes.NewValidatorHistoricalRewards(sdk.NewDecCoins(sdk.NewDecCoin("udo", math.NewInt(1))), 2))
	s.App.DistrKeeper.SetValidatorCurrentRewards(s.Ctx, valAddrs[0], disttypes.NewValidatorCurrentRewards(sdk.NewDecCoins(sdk.NewDecCoin("udo", math.NewInt(1))), 2))
	s.App.DistrKeeper.SetDelegatorStartingInfo(s.Ctx, valAddrs[0], s.TestAccs[0], disttypes.NewDelegatorStartingInfo(1, math.LegacyOneDec(), 1))
	// first delegation should be normal
	// raise voting power of validator 0 by 1 (1+1)/(10+1) = 0.181818 < 0.2
	s.App.StakingKeeper.SetDelegation(s.Ctx, stakingtypes.NewDelegation(s.TestAccs[0].String(), valAddrs[0].String(), math.LegacyNewDec(1000000)))
	_, err := s.App.StakingKeeper.Delegate(s.Ctx, s.TestAccs[0], math.NewInt(1000000), stakingtypes.Unbonded, validators[0], true)
	s.Require().NoError(err)

	// update validator set and validator 0 state
	_, err = s.App.StakingKeeper.ApplyAndReturnValidatorSetUpdates(s.Ctx)
	s.Require().NoError(err)
	validator, err := s.App.StakingKeeper.GetValidator(s.Ctx, valAddrs[0])
	s.Require().NoError(err)
	validators[0] = validator

	s.App.StakingKeeper.SetDelegation(s.Ctx, stakingtypes.NewDelegation(s.TestAccs[0].String(), valAddrs[0].String(), math.LegacyNewDec(1000000)))
	_, err = s.App.StakingKeeper.Delegate(s.Ctx, s.TestAccs[0], math.NewInt(1000000), stakingtypes.Unbonded, validators[0], true)
	// Assert that an error was returned
	s.Require().Error(err, fmt.Sprintf("voting power is %v, should be > 20", validators[0].ConsensusPower(s.App.StakingKeeper.PowerReduction(s.Ctx))))
	s.Require().Equal("validator power is over the allowed limit", err.Error())
}

func (s *StakingTestSuite) TestEqualConsensusPowerOnlyWhenValidatorSetChanges() {
	s.Setup(s.T(), types.ColumbusChainID)

	num := 2
	addrDels := s.RandomAccountAddresses(num)
	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrDels)
	PKs := simtestutil.CreateTestPubKeys(num)
	amts := []math.Int{math.NewInt(1_000_000), math.NewInt(9_000_000)}

	for i, addrDel := range addrDels {
		s.FundAcc(addrDel, sdk.NewCoins(sdk.NewInt64Coin("udo", amts[i].Int64())))
		err := s.App.BankKeeper.DelegateCoinsFromAccountToModule(s.Ctx, addrDel, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin("udo", amts[i])))
		s.Require().NoError(err)

		validator := testutil.NewValidator(s.T(), valAddrs[i], PKs[i])
		validator, _ = validator.AddTokensFromDel(amts[i])
		stakingkeeper.TestingUpdateValidator(s.App.StakingKeeper, s.Ctx, validator, true)
	}

	module := customstaking.NewAppModule(
		s.App.AppCodec(),
		s.App.StakingKeeper,
		s.App.AccountKeeper,
		s.App.BankKeeper,
		s.App.ParamsKeeper,
		s.App.GetSubspace(stakingtypes.ModuleName),
	)

	updates, err := module.EndBlock(s.Ctx)
	s.Require().NoError(err)
	// TestingUpdateValidator(..., true) has already applied the validator set,
	// so EndBlock must not resend unchanged validators to CometBFT.
	s.Require().Empty(updates)

	storedUpdates, err := s.App.StakingKeeper.GetValidatorUpdates(s.Ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(storedUpdates)
	for _, update := range storedUpdates {
		if update.Power > 0 {
			s.Require().Equal(customstaking.EqualValidatorConsensusPower, update.Power)
		}
	}

	hasStakeWeightedPower := false
	err = s.App.StakingKeeper.IterateLastValidatorPowers(s.Ctx, func(_ sdk.ValAddress, power int64) (stop bool) {
		if power > customstaking.EqualValidatorConsensusPower {
			hasStakeWeightedPower = true
		}
		return false
	})
	s.Require().NoError(err)
	s.Require().True(hasStakeWeightedPower)

	// Queue one additional validator without applying it. The next EndBlock must
	// emit the change and must normalize its consensus power to one.
	newAddrDel := s.RandomAccountAddresses(1)[0]
	newValAddr := sdk.ValAddress(newAddrDel)
	newAmount := math.NewInt(2_000_000)
	s.FundAcc(newAddrDel, sdk.NewCoins(sdk.NewCoin("udo", newAmount)))
	err = s.App.BankKeeper.DelegateCoinsFromAccountToModule(
		s.Ctx,
		newAddrDel,
		stakingtypes.NotBondedPoolName,
		sdk.NewCoins(sdk.NewCoin("udo", newAmount)),
	)
	s.Require().NoError(err)
	newValidator := testutil.NewValidator(s.T(), newValAddr, simtestutil.CreateTestPubKeys(1)[0])
	newValidator, _ = newValidator.AddTokensFromDel(newAmount)
	stakingkeeper.TestingUpdateValidator(s.App.StakingKeeper, s.Ctx, newValidator, false)

	updates, err = module.EndBlock(s.Ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(updates)
	for _, update := range updates {
		if update.Power > 0 {
			s.Require().Equal(customstaking.EqualValidatorConsensusPower, update.Power)
		}
	}

	updates, err = module.EndBlock(s.Ctx)
	s.Require().NoError(err)
	s.Require().Empty(updates)
}
