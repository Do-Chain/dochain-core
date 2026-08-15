package post

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"
)

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
}

type fakeFeeTx struct {
	msgs       []sdk.Msg
	gas        uint64
	fee        sdk.Coins
	feePayer   sdk.AccAddress
	feeGranter sdk.AccAddress
}

func (tx fakeFeeTx) GetMsgs() []sdk.Msg {
	return tx.msgs
}

func (tx fakeFeeTx) GetMsgsV2() ([]protov2.Message, error) {
	return nil, nil
}

func (tx fakeFeeTx) GetGas() uint64 {
	return tx.gas
}

func (tx fakeFeeTx) GetFee() sdk.Coins {
	return tx.fee
}

func (tx fakeFeeTx) FeePayer() []byte {
	return tx.feePayer
}

func (tx fakeFeeTx) FeeGranter() []byte {
	return tx.feeGranter
}

type fakeAccountKeeper struct {
	accounts map[string]sdk.AccountI
}

func (ak fakeAccountKeeper) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	if ak.accounts == nil {
		return nil
	}
	return ak.accounts[addr.String()]
}

type fakeBankKeeper struct {
	refund sdk.Coins
	to     sdk.AccAddress
}

func (bk *fakeBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, recipient sdk.AccAddress, amt sdk.Coins) error {
	bk.to = recipient
	bk.refund = amt
	return nil
}

type fakeValueFeeKeeper struct {
	params valuefeetypes.Params
	calls  int
}

func (vk *fakeValueFeeKeeper) GetParams(sdk.Context) valuefeetypes.Params {
	vk.calls++
	return vk.params
}

func TestRefundUnusedGasDecoratorDoesNothingBeforeActivationHeight(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer---------------"))
	valueKeeper := &fakeValueFeeKeeper{params: valuefeetypes.MainnetV23Params()}
	decorator := NewRefundUnusedGasDecorator(fakeAccountKeeper{}, &fakeBankKeeper{}, valueKeeper, 100, 0)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(99)
	ctx.GasMeter().ConsumeGas(100, "existing")
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{testdata.NewTestMsg(payer)},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000)),
		feePayer: payer,
	}

	newCtx, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.Equal(t, uint64(100), newCtx.GasMeter().GasConsumed())
	require.Zero(t, valueKeeper.calls)
}

func TestRefundUnusedGasDecoratorRefundsSuccessfulNormalTxAfterActivation(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer---------------"))
	valueKeeper := &fakeValueFeeKeeper{params: valuefeetypes.MainnetV23Params()}
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(fakeAccountKeeper{}, bankKeeper, valueKeeper, 100, 0)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(100)
	ctx.GasMeter().ConsumeGas(240, "test")
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{testdata.NewTestMsg(payer)},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000)),
		feePayer: payer,
	}

	_, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, valueKeeper.calls)
	require.Equal(t, payer, bankKeeper.to)
	require.Equal(t, "600000000udo", bankKeeper.refund.String())
}

func TestRefundUnusedGasDecoratorSkipsFailedTx(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer---------------"))
	valueKeeper := &fakeValueFeeKeeper{params: valuefeetypes.MainnetV23Params()}
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(fakeAccountKeeper{}, bankKeeper, valueKeeper, 100, 0)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(100)
	ctx.GasMeter().ConsumeGas(100, "test")
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{testdata.NewTestMsg(payer)},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000)),
		feePayer: payer,
	}

	_, err := decorator.PostHandle(ctx, tx, false, false, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.True(t, bankKeeper.refund.IsZero())
	require.Zero(t, valueKeeper.calls)
}

func TestRefundUnusedGasDecoratorSkipsPureValueSend(t *testing.T) {
	from := sdk.AccAddress([]byte("from----------------"))
	to := sdk.AccAddress([]byte("to------------------"))
	valueKeeper := &fakeValueFeeKeeper{params: valuefeetypes.MainnetV23Params()}
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(
		fakeAccountKeeper{
			accounts: map[string]sdk.AccountI{
				from.String(): authtypes.NewBaseAccountWithAddress(from),
				to.String():   authtypes.NewBaseAccountWithAddress(to),
			},
		},
		bankKeeper,
		valueKeeper,
		100,
		0,
	)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(100)
	ctx.GasMeter().ConsumeGas(100, "test")
	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000*core.MicroUnit)))
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{msg},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000*core.MicroUnit)),
		feePayer: from,
	}

	_, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.True(t, bankKeeper.refund.IsZero())
}

func TestRefundUnusedGasDecoratorRefundsOnlyNormalFeeWhenMixedTxIncludesValueFee(t *testing.T) {
	from := sdk.AccAddress([]byte("from----------------"))
	to := sdk.AccAddress([]byte("to------------------"))
	payer := sdk.AccAddress([]byte("payer---------------"))
	valueKeeper := &fakeValueFeeKeeper{params: valuefeetypes.MainnetV23Params()}
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(
		fakeAccountKeeper{
			accounts: map[string]sdk.AccountI{
				from.String(): authtypes.NewBaseAccountWithAddress(from),
				to.String():   authtypes.NewBaseAccountWithAddress(to),
			},
		},
		bankKeeper,
		valueKeeper,
		100,
		0,
	)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(100)
	ctx.GasMeter().ConsumeGas(150, "test")
	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1_000*core.MicroUnit)))
	tx := fakeFeeTx{
		msgs: []sdk.Msg{
			msg,
			testdata.NewTestMsg(payer),
		},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 4_000_000_000)),
		feePayer: payer,
	}

	_, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.Equal(t, "1500000000udo", bankKeeper.refund.String())
}

func TestRefundUnusedGasDecoratorSkipsFeeExemptTx(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer---------------"))
	params := valuefeetypes.MainnetV24Params()
	params.FeeExemptAddresses = []string{payer.String()}
	valueKeeper := &fakeValueFeeKeeper{params: params}
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(fakeAccountKeeper{}, bankKeeper, valueKeeper, 100, 100)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(100)
	ctx.GasMeter().ConsumeGas(150, "test")
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{testdata.NewTestMsg(payer)},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000)),
		feePayer: payer,
	}

	_, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.True(t, bankKeeper.refund.IsZero())
}

func TestRefundUnusedGasDecoratorDoesNotSkipFeeExemptBeforeV24(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer---------------"))
	params := valuefeetypes.MainnetV24Params()
	params.FeeExemptAddresses = []string{payer.String()}
	valueKeeper := &fakeValueFeeKeeper{params: params}
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(fakeAccountKeeper{}, bankKeeper, valueKeeper, 100, 200)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(100)
	ctx.GasMeter().ConsumeGas(150, "test")
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{testdata.NewTestMsg(payer)},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000)),
		feePayer: payer,
	}

	_, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.Equal(t, "1500000000udo", bankKeeper.refund.String())
}

func TestIsFeeExemptTxUsesV25MainnetFallback(t *testing.T) {
	payer, err := sdk.AccAddressFromBech32("do16w707l5t2ru9xuhjguc2zcf59845j0urt5c0r0")
	require.NoError(t, err)

	params := valuefeetypes.MainnetV25Params()
	params.FeeExemptAddresses = nil
	exempt, err := isFeeExemptTx(fakeFeeTx{feePayer: payer}, params)

	require.NoError(t, err)
	require.True(t, exempt)
}

func TestIsFeeExemptTxDoesNotUseV25FallbackBeforeV25(t *testing.T) {
	payer, err := sdk.AccAddressFromBech32("do16w707l5t2ru9xuhjguc2zcf59845j0urt5c0r0")
	require.NoError(t, err)

	params := valuefeetypes.MainnetV24Params()
	params.FeeExemptAddresses = nil
	exempt, err := isFeeExemptTx(fakeFeeTx{feePayer: payer}, params)

	require.NoError(t, err)
	require.False(t, exempt)
}
