package post

import (
	"bytes"
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/app/helper"
	core "github.com/Daviddochain/dochain-core/v4/types"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type ValueFeeKeeper interface {
	GetParams(ctx sdk.Context) valuefeetypes.Params
}

type RefundUnusedGasDecorator struct {
	accountKeeper    AccountKeeper
	bankKeeper       BankKeeper
	valueFeeKeeper   ValueFeeKeeper
	activationHeight int64
}

func NewRefundUnusedGasDecorator(ak AccountKeeper, bk BankKeeper, vk ValueFeeKeeper, activationHeight int64) RefundUnusedGasDecorator {
	return RefundUnusedGasDecorator{
		accountKeeper:    ak,
		bankKeeper:       bk,
		valueFeeKeeper:   vk,
		activationHeight: activationHeight,
	}
}

func (rd RefundUnusedGasDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {
	newCtx, err := next(ctx, tx, simulate, success)
	if err != nil {
		return newCtx, err
	}

	if rd.activationHeight <= 0 || newCtx.BlockHeight() < rd.activationHeight {
		return newCtx, nil
	}
	if simulate || newCtx.IsCheckTx() || !success || rd.valueFeeKeeper == nil {
		return newCtx, nil
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return newCtx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	msgs := feeTx.GetMsgs()
	if helper.IsOracleTx(msgs) {
		return newCtx, nil
	}

	params := rd.valueFeeKeeper.GetParams(newCtx)
	if params.PolicyVersion < 23 {
		return newCtx, nil
	}
	if err := params.Validate(); err != nil {
		return newCtx, err
	}

	valueFee, hasValueFee, pureValueSend, err := rd.computeValueSendFee(newCtx, msgs, params)
	if err != nil {
		return newCtx, err
	}
	if pureValueSend {
		return newCtx, nil
	}

	gasLimit := feeTx.GetGas()
	gasConsumed := newCtx.GasMeter().GasConsumed()
	if gasLimit == 0 || gasConsumed >= gasLimit {
		return newCtx, nil
	}

	paidNormalFee := feeTx.GetFee().AmountOf(core.MicroDoDenom)
	if hasValueFee {
		if paidNormalFee.LTE(valueFee.Amount) {
			return newCtx, nil
		}
		paidNormalFee = paidNormalFee.Sub(valueFee.Amount)
	}
	refundAmount := paidNormalFee.
		Mul(sdkmath.NewIntFromUint64(gasLimit - gasConsumed)).
		Quo(sdkmath.NewIntFromUint64(gasLimit))
	if !refundAmount.IsPositive() {
		return newCtx, nil
	}

	feePayer, err := feeDeductedFrom(feeTx)
	if err != nil {
		return newCtx, err
	}
	refund := sdk.NewCoins(sdk.NewCoin(core.MicroDoDenom, refundAmount))
	if err := rd.bankKeeper.SendCoinsFromModuleToAccount(sdk.WrapSDKContext(newCtx), authtypes.FeeCollectorName, feePayer, refund); err != nil {
		return newCtx, err
	}

	newCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"unused_gas_refund",
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, feePayer.String()),
			sdk.NewAttribute("refund", refund.String()),
			sdk.NewAttribute("gas_limit", fmt.Sprintf("%d", gasLimit)),
			sdk.NewAttribute("gas_used", fmt.Sprintf("%d", gasConsumed)),
		),
	)

	return newCtx, nil
}

func feeDeductedFrom(feeTx sdk.FeeTx) (sdk.AccAddress, error) {
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	if len(feeGranter) != 0 && !bytes.Equal(feeGranter, feePayer) {
		return sdk.AccAddress(feeGranter), nil
	}
	if len(feePayer) != 0 {
		return sdk.AccAddress(feePayer), nil
	}

	sigTx, ok := feeTx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, fmt.Errorf("fee payer address not found and cannot cast to SigVerifiableTx")
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return nil, fmt.Errorf("fee payer address not found and cannot get signers: %w", err)
	}
	if len(signers) == 0 {
		return nil, fmt.Errorf("fee payer address not found and no signers available")
	}
	return signers[0], nil
}

func (rd RefundUnusedGasDecorator) computeValueSendFee(ctx sdk.Context, msgs []sdk.Msg, params valuefeetypes.Params) (sdk.Coin, bool, bool, error) {
	zero := sdk.NewCoin(core.MicroDoDenom, sdkmath.ZeroInt())
	if !params.Enabled || !params.ApplyToMsgSend || params.PolicyVersion == 0 {
		return zero, false, false, nil
	}

	totalValue := sdkmath.ZeroInt()
	eligibleMsgs := 0
	pureValueSend := len(msgs) > 0
	for _, msg := range msgs {
		sendMsg, ok := msg.(*banktypes.MsgSend)
		if !ok {
			pureValueSend = false
			continue
		}
		fromAddr, err := sdk.AccAddressFromBech32(sendMsg.FromAddress)
		if err != nil {
			return zero, false, false, err
		}
		toAddr, err := sdk.AccAddressFromBech32(sendMsg.ToAddress)
		if err != nil {
			return zero, false, false, err
		}
		if rd.isModuleAccount(ctx, fromAddr) || rd.isModuleAccount(ctx, toAddr) {
			pureValueSend = false
			continue
		}

		msgHasValueFee := false
		for _, coin := range sendMsg.Amount {
			value, known := denomValueInFeeDenom(coin, params)
			if known {
				totalValue = totalValue.Add(value)
				msgHasValueFee = true
				continue
			}
			if params.ChargeUnknownDenomMinFee {
				msgHasValueFee = true
				continue
			}
			pureValueSend = false
		}
		if !msgHasValueFee {
			pureValueSend = false
			continue
		}
		eligibleMsgs++
	}

	if eligibleMsgs == 0 {
		return zero, false, false, nil
	}

	feeAmount := totalValue.MulRaw(int64(params.RateBps)).QuoRaw(10_000)
	if feeAmount.LT(params.MinFee) {
		feeAmount = params.MinFee
	}
	if params.MaxFeeEnabled && feeAmount.GT(params.MaxFee) {
		feeAmount = params.MaxFee
	}

	return sdk.NewCoin(params.FeeDenom, feeAmount), true, pureValueSend, nil
}

func (rd RefundUnusedGasDecorator) isModuleAccount(ctx sdk.Context, addr sdk.AccAddress) bool {
	if rd.accountKeeper == nil {
		return false
	}
	acc := rd.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return false
	}
	_, ok := acc.(authtypes.ModuleAccountI)
	return ok
}

func denomValueInFeeDenom(coin sdk.Coin, params valuefeetypes.Params) (sdkmath.Int, bool) {
	for _, rate := range params.DenomValueRates {
		if rate.Denom == coin.Denom {
			return coin.Amount.Mul(rate.UdoPerBaseUnit), true
		}
	}
	return sdkmath.ZeroInt(), false
}
