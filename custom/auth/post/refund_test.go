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
	moduleAddr sdk.AccAddress
	accounts   map[string]sdk.AccountI
}

func (ak fakeAccountKeeper) GetModuleAddress(string) sdk.AccAddress {
	return ak.moduleAddr
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
}

func (vk fakeValueFeeKeeper) GetParams(sdk.Context) valuefeetypes.Params {
	return vk.params
}

func TestRefundableGasFeesRefundsOnlyUnusedNormalGas(t *testing.T) {
	paid := sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000))
	limit := sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000))
	used := sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2_400_000_000))

	refund := refundableGasFees(paid, limit, used)

	require.Equal(t, "600000000udo", refund.String())
}

func TestRefundUnusedGasDecoratorRefundsSuccessfulNormalTx(t *testing.T) {
	payer := sdk.AccAddress([]byte("payer---------------"))
	params := valuefeetypes.MainnetV23Params()
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(
		fakeAccountKeeper{moduleAddr: sdk.AccAddress([]byte("fee-collector-------"))},
		bankKeeper,
		fakeValueFeeKeeper{params: params},
	)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(1)
	ctx.GasMeter().ConsumeGas(240, "test")
	msg := testdata.NewTestMsg(payer)
	tx := fakeFeeTx{
		msgs:     []sdk.Msg{msg},
		gas:      300,
		fee:      sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 3_000_000_000)),
		feePayer: payer,
	}

	_, err := decorator.PostHandle(ctx, tx, false, true, func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
		return ctx, nil
	})

	require.NoError(t, err)
	require.Equal(t, payer, bankKeeper.to)
	require.Equal(t, "600000000udo", bankKeeper.refund.String())
}

func TestRefundUnusedGasDecoratorSkipsPureValueSend(t *testing.T) {
	from := sdk.AccAddress([]byte("from----------------"))
	to := sdk.AccAddress([]byte("to------------------"))
	params := valuefeetypes.MainnetV23Params()
	bankKeeper := &fakeBankKeeper{}
	decorator := NewRefundUnusedGasDecorator(
		fakeAccountKeeper{
			moduleAddr: sdk.AccAddress([]byte("fee-collector-------")),
			accounts: map[string]sdk.AccountI{
				from.String(): authtypes.NewBaseAccountWithAddress(from),
				to.String():   authtypes.NewBaseAccountWithAddress(to),
			},
		},
		bankKeeper,
		fakeValueFeeKeeper{params: params},
	)
	ctx := sdk.Context{}.
		WithGasMeter(storetypes.NewGasMeter(300)).
		WithEventManager(sdk.NewEventManager()).
		WithBlockHeight(1)
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
