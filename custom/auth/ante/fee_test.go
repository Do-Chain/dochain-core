package ante_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/custom/auth/ante"
	core "github.com/Daviddochain/dochain-core/v4/types"
	oracletypes "github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *AnteTestSuite) feeHandler() sdk.AnteHandler {
	decorator := ante.NewFeeDecorator(
		s.app.AccountKeeper,
		s.app.BankKeeper,
		s.app.FeeGrantKeeper,
		s.app.TreasuryKeeper,
		s.app.DistrKeeper,
		s.app.ValueFeeKeeper,
	)
	return sdk.ChainAnteDecorators(decorator)
}

func (s *AnteTestSuite) buildSignedTx(priv cryptotypes.PrivKey, msgs ...sdk.Msg) sdk.Tx {
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msgs...))
	tx, err := s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)
	return tx
}

func (s *AnteTestSuite) TestFeeDecoratorRejectsZeroGasOutsideSimulation() {
	s.SetupTest(true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	priv, _, address := testdata.KeyTestPubAddr()
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(300))),
	))
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(address)))
	s.txBuilder.SetGasLimit(0)

	tx, err := s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)

	_, err = s.feeHandler()(s.ctx.WithIsCheckTx(true), tx, false)
	s.Require().Error(err)
	_, err = s.feeHandler()(s.ctx.WithIsCheckTx(true), tx, true)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestFeeDecoratorEnforcesMempoolMinimumAndPriority() {
	s.SetupTest(true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	priv, _, address := testdata.KeyTestPubAddr()
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(300))),
	))
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(address)))
	s.txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	s.txBuilder.SetGasLimit(15)

	tx, err := s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)

	highPrice := sdk.NewDecCoins(sdk.NewDecCoinFromDec("atom", sdkmath.LegacyNewDec(20)))
	_, err = s.feeHandler()(s.ctx.WithIsCheckTx(true).WithMinGasPrices(highPrice), tx, false)
	s.Require().Error(err)

	lowPrice := sdk.NewDecCoins(
		sdk.NewDecCoinFromDec("atom", sdkmath.LegacyNewDec(1).QuoInt64(100000)),
	)
	newCtx, err := s.feeHandler()(s.ctx.WithIsCheckTx(true).WithMinGasPrices(lowPrice), tx, false)
	s.Require().NoError(err)
	s.Require().Equal(int64(10), newCtx.Priority())
}

func (s *AnteTestSuite) TestFeeDecoratorUsesChainGasPriceAfterV23() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV23Params())
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	priv, _, address := testdata.KeyTestPubAddr()
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2_000_000_000)),
	))
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(address)))
	s.txBuilder.SetGasLimit(100)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 999_999_999)))

	tx, err := s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)

	lowNodePrice := sdk.NewDecCoins(sdk.NewDecCoin(core.MicroDoDenom, sdkmath.OneInt()))
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(lowNodePrice), tx, false)
	s.Require().Error(err)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(address)))
	s.txBuilder.SetGasLimit(100)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000_000)))
	tx, err = s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(lowNodePrice), tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestFeeDecoratorDeductsOnlyConfiguredFee() {
	s.SetupTest(true)
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	priv, _, address := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, address)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(10))),
	))
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(address)))
	s.txBuilder.SetFeeAmount(testdata.NewTestFeeAmount())
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())

	tx, err := s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().Error(err)

	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(200))),
	))
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().NoError(err)
}

func TestRemovedTaxModuleCannotAddHiddenTransferFees(t *testing.T) {
	from := sdk.AccAddress(make([]byte, 20))
	to := sdk.AccAddress(append(make([]byte, 19), 1))
	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("udo", 1_000_000)))
	taxes, nonTaxable := ante.FilterMsgAndComputeTax(sdk.Context{}, msg)
	if !taxes.Empty() || !nonTaxable.Empty() {
		t.Fatalf("removed tax module returned charges: taxable=%s non-taxable=%s", taxes, nonTaxable)
	}
}

func (s *AnteTestSuite) TestOracleMessagesRemainZeroFee() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV21Params())
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	priv, _, address := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, address)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 1_000_000_000)),
	))
	validator, err := stakingtypes.NewValidator(
		sdk.ValAddress(address).String(), priv.PubKey(), stakingtypes.Description{},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.app.StakingKeeper.SetValidator(s.ctx, validator))

	msg := oracletypes.NewMsgAggregateDoRatePrevote(
		oracletypes.GetAggregateVoteHash("salt", "exchange rates", sdk.ValAddress(validator.GetOperator())),
		address,
		sdk.ValAddress(validator.GetOperator()),
	)
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroSDRDenom, 12345)))
	tx, err := s.CreateTestTx(
		[]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID(),
	)
	s.Require().NoError(err)

	before := s.app.BankKeeper.GetAllBalances(s.ctx, address)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().NoError(err)
	after := s.app.BankKeeper.GetAllBalances(s.ctx, address)
	s.Require().Equal(before, after)
	s.Require().Empty(s.app.BankKeeper.GetAllBalances(
		s.ctx, s.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName),
	))
}

func (s *AnteTestSuite) TestDirectDoSendUsesMinimumValueFeeOnly() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV21Params())

	priv, _, from := testdata.KeyTestPubAddr()
	_, _, to := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, from)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		from,
		sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2_000_000_000)),
	))

	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 100*core.MicroUnit)))

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 999_999_999)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(core.MicroDoDenom, sdkmath.NewInt(10_000_000)))), tx, false)
	s.Require().Error(err)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000_000)))
	tx, err = s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(core.MicroDoDenom, sdkmath.NewInt(10_000_000)))), tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestLargeDirectDoSendUsesRateFee() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV21Params())

	priv, _, from := testdata.KeyTestPubAddr()
	_, _, to := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, from)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		from,
		sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2_000_000_000_000)),
	))

	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 10_000_000_000*core.MicroUnit)))
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000_000_000)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)

	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestDirectDODxSendPaysMinimumFeeInDo() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV22Params())

	priv, _, from := testdata.KeyTestPubAddr()
	_, _, to := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, from)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		from,
		sdk.NewCoins(
			sdk.NewInt64Coin(core.MicroDoDenom, 2_000_000_000),
			sdk.NewInt64Coin(core.MicroDODxDenom, 200*core.MicroUnit),
		),
	))

	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDODxDenom, 100*core.MicroUnit)))

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDODxDenom, 1_000_000_000)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().Error(err)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000_000)))
	tx, err = s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestLargeDirectDODxSendUsesDoValueRateFee() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV22Params())

	priv, _, from := testdata.KeyTestPubAddr()
	_, _, to := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, from)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		from,
		sdk.NewCoins(
			sdk.NewInt64Coin(core.MicroDoDenom, 2_000_000_000_000),
			sdk.NewInt64Coin(core.MicroDODxDenom, 10_000_000_000*core.MicroUnit),
		),
	))

	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDODxDenom, 10_000_000_000*core.MicroUnit)))

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 999_999_999_999)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().Error(err)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000_000_000)))
	tx, err = s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestUnknownDirectCoinSendPaysMinimumDoFeeAfterV22() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV22Params())

	priv, _, from := testdata.KeyTestPubAddr()
	_, _, to := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, from)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		from,
		sdk.NewCoins(
			sdk.NewInt64Coin(core.MicroDoDenom, 2_000_000_000),
			sdk.NewInt64Coin("ugame", 1_000_000_000_000),
		),
	))

	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("ugame", 1_000_000_000_000)))

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 999_999_999)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().Error(err)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(testdata.NewTestGasLimit())
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000_000)))
	tx, err = s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx, tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestUnknownDirectCoinSendKeepsNormalGasBeforeV22() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV21Params())

	priv, _, from := testdata.KeyTestPubAddr()
	_, _, to := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, from)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		from,
		sdk.NewCoins(
			sdk.NewInt64Coin(core.MicroDoDenom, 10_000),
			sdk.NewInt64Coin("ugame", 1_000_000),
		),
	))

	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("ugame", 1_000_000)))

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetGasLimit(1000)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1000)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(core.MicroDoDenom, sdkmath.OneInt()))), tx, false)
	s.Require().NoError(err)
}

func (s *AnteTestSuite) TestValueFeeDoesNotApplyToGovernanceOrStaking() {
	s.SetupTest(true)
	s.app.ValueFeeKeeper.SetParams(s.ctx, valuefeetypes.MainnetV21Params())

	priv, _, address := testdata.KeyTestPubAddr()
	account := s.app.AccountKeeper.NewAccountWithAddress(s.ctx, address)
	s.app.AccountKeeper.SetAccount(s.ctx, account)
	s.Require().NoError(banktestutil.FundAccount(
		s.ctx,
		s.app.BankKeeper,
		address,
		sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 10_000_000)),
	))

	vote := govtypes.NewMsgVote(address, 1, govtypes.OptionYes, "")
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(vote))
	s.txBuilder.SetGasLimit(1000)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1000)))
	tx, err := s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(core.MicroDoDenom, sdkmath.OneInt()))), tx, false)
	s.Require().NoError(err)

	delegate := stakingtypes.NewMsgDelegate(address.String(), sdk.ValAddress(address).String(), sdk.NewInt64Coin(core.MicroDoDenom, 1_000_000))
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	s.Require().NoError(s.txBuilder.SetMsgs(delegate))
	s.txBuilder.SetGasLimit(1000)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1000)))
	tx, err = s.CreateTestTx([]cryptotypes.PrivKey{priv}, []uint64{0}, []uint64{0}, s.ctx.ChainID())
	s.Require().NoError(err)
	_, err = s.feeHandler()(s.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(core.MicroDoDenom, sdkmath.OneInt()))), tx, false)
	s.Require().NoError(err)
}
