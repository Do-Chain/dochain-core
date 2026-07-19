package simulation

// DONTCOVER

import (
	"math/rand"
	"strings"

	"cosmossdk.io/math"
	appparams "github.com/Daviddochain/dochain-core/v4/app/params"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/oracle/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	banksim "github.com/cosmos/cosmos-sdk/x/bank/simulation"
	distrsim "github.com/cosmos/cosmos-sdk/x/distribution/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// Simulation operation weights constants.
const (
	OpWeightMsgAggregateDoRatePrevote = "op_weight_msg_exchange_rate_aggregate_prevote" // #nosec
	OpWeightMsgAggregateDoRateVote    = "op_weight_msg_exchange_rate_aggregate_vote"    // #nosec
	OpWeightMsgDelegateFeedConsent    = "op_weight_msg_exchange_feed_consent"           // #nosec

	salt = "1234"
)

var (
	// Keep the simulation whitelist aligned with the chain's base denom defaults.
	whitelist   = []string{core.MicroDoDenom}
	voteHashMap = make(map[string]string)
)

// WeightedOperations returns all the operations from the module with their respective weights.
func WeightedOperations(
	appParams simtypes.AppParams,
	cdc codec.JSONCodec,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simulation.WeightedOperations {
	var (
		weightMsgAggregateDoRatePrevote int
		weightMsgAggregateDoRateVote    int
		weightMsgDelegateFeedConsent    int
	)

	appParams.GetOrGenerate(OpWeightMsgAggregateDoRatePrevote, &weightMsgAggregateDoRatePrevote, nil,
		func(*rand.Rand) {
			weightMsgAggregateDoRatePrevote = banksim.DefaultWeightMsgSend * 2
		},
	)

	appParams.GetOrGenerate(OpWeightMsgAggregateDoRateVote, &weightMsgAggregateDoRateVote, nil,
		func(*rand.Rand) {
			weightMsgAggregateDoRateVote = banksim.DefaultWeightMsgSend * 2
		},
	)

	appParams.GetOrGenerate(OpWeightMsgDelegateFeedConsent, &weightMsgDelegateFeedConsent, nil,
		func(*rand.Rand) {
			weightMsgDelegateFeedConsent = distrsim.DefaultWeightMsgSetWithdrawAddress
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgAggregateDoRatePrevote,
			SimulateMsgAggregateDoRatePrevote(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightMsgAggregateDoRateVote,
			SimulateMsgAggregateDoRateVote(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightMsgDelegateFeedConsent,
			SimulateMsgDelegateFeedConsent(ak, bk, k),
		),
	}
}

// SimulateMsgAggregateDoRatePrevote generates a MsgAggregateDoRatePrevote with random values.
func SimulateMsgAggregateDoRatePrevote(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		address := sdk.ValAddress(simAccount.Address)

		// Ensure the validator exists.
		val, err := k.StakingKeeper.Validator(ctx, address)
		if err != nil || val == nil || !val.IsBonded() {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRatePrevote, "unable to find validator"), nil, nil
		}

		exchangeRatesStr := ""
		for _, denom := range whitelist {
			price := math.LegacyNewDecWithPrec(int64(simtypes.RandIntBetween(r, 1, 10000)), 1)
			exchangeRatesStr += price.String() + denom + ","
		}

		exchangeRatesStr = strings.TrimRight(exchangeRatesStr, ",")
		voteHash := types.GetAggregateVoteHash(salt, exchangeRatesStr, address)

		feederAddr := k.GetFeederDelegation(ctx, address)
		feederSimAccount, _ := simtypes.FindAccount(accs, feederAddr)

		feederAccount := ak.GetAccount(ctx, feederAddr)
		spendable := bk.SpendableCoins(ctx, feederAccount.GetAddress())

		fees, err := simtypes.RandomFees(r, ctx, spendable)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRatePrevote, "unable to generate fees"), nil, err
		}

		msg := types.NewMsgAggregateDoRatePrevote(voteHash, feederAddr, address)

		txGen := appparams.MakeSimulationTxConfig()
		tx, err := simtestutil.GenSignedMockTx(
			r,
			txGen,
			[]sdk.Msg{msg},
			fees,
			simtestutil.DefaultGenTxGas,
			chainID,
			[]uint64{feederAccount.GetAccountNumber()},
			[]uint64{feederAccount.GetSequence()},
			feederSimAccount.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to generate mock tx"), nil, err
		}

		_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to deliver tx"), nil, err
		}

		voteHashMap[address.String()] = exchangeRatesStr

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgAggregateDoRateVote generates a MsgAggregateDoRateVote with random values.
func SimulateMsgAggregateDoRateVote(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		address := sdk.ValAddress(simAccount.Address)

		// Ensure the validator exists.
		val, err := k.StakingKeeper.Validator(ctx, address)
		if err != nil || val == nil || !val.IsBonded() {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRateVote, "unable to find validator"), nil, nil
		}

		// Ensure vote hash exists.
		exchangeRatesStr, ok := voteHashMap[address.String()]
		if !ok {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRateVote, "vote hash not exists"), nil, nil
		}

		// Get prevote.
		prevote, err := k.GetAggregateDoRatePrevote(ctx, address)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRateVote, "prevote not found"), nil, nil
		}

		params := k.GetParams(ctx)
		if (uint64(ctx.BlockHeight())/params.VotePeriod)-(prevote.SubmitBlock/params.VotePeriod) != 1 {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRateVote, "reveal period of submitted vote does not match registered prevote"), nil, nil
		}

		feederAddr := k.GetFeederDelegation(ctx, address)
		feederSimAccount, _ := simtypes.FindAccount(accs, feederAddr)
		feederAccount := ak.GetAccount(ctx, feederAddr)
		spendableCoins := bk.SpendableCoins(ctx, feederAddr)

		fees, err := simtypes.RandomFees(r, ctx, spendableCoins)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRateVote, "unable to generate fees"), nil, err
		}

		msg := types.NewMsgAggregateDoRateVote(salt, exchangeRatesStr, feederAddr, address)

		txGen := appparams.MakeSimulationTxConfig()
		tx, err := simtestutil.GenSignedMockTx(
			r,
			txGen,
			[]sdk.Msg{msg},
			fees,
			simtestutil.DefaultGenTxGas,
			chainID,
			[]uint64{feederAccount.GetAccountNumber()},
			[]uint64{feederAccount.GetSequence()},
			feederSimAccount.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to generate mock tx"), nil, err
		}

		_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to deliver tx"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgDelegateFeedConsent generates a MsgDelegateFeedConsent with random values.
func SimulateMsgDelegateFeedConsent(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		delegateAccount, _ := simtypes.RandomAcc(r, accs)
		valAddress := sdk.ValAddress(simAccount.Address)
		delegateValAddress := sdk.ValAddress(delegateAccount.Address)
		account := ak.GetAccount(ctx, simAccount.Address)

		// Ensure the validator exists.
		val, err := k.StakingKeeper.Validator(ctx, valAddress)
		if err != nil || val == nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgDelegateFeedConsent, "unable to find validator"), nil, nil
		}

		// Ensure the target address is not a validator.
		val2, err := k.StakingKeeper.Validator(ctx, delegateValAddress)
		if err != nil || val2 != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgDelegateFeedConsent, "unable to delegate to validator"), nil, nil
		}

		spendableCoins := bk.SpendableCoins(ctx, account.GetAddress())
		fees, err := simtypes.RandomFees(r, ctx, spendableCoins)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, types.TypeMsgAggregateDoRateVote, "unable to generate fees"), nil, err
		}

		msg := types.NewMsgDelegateFeedConsent(valAddress, delegateAccount.Address)

		txGen := appparams.MakeSimulationTxConfig()
		tx, err := simtestutil.GenSignedMockTx(
			r,
			txGen,
			[]sdk.Msg{msg},
			fees,
			simtestutil.DefaultGenTxGas,
			chainID,
			[]uint64{account.GetAccountNumber()},
			[]uint64{account.GetSequence()},
			simAccount.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to generate mock tx"), nil, err
		}

		_, _, err = app.SimDeliver(txGen.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "unable to deliver tx"), nil, err
		}

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}
