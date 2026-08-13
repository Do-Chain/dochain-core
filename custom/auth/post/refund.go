package post

import (
	"bytes"
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/app/helper"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type ValueFeeKeeper interface {
	GetParams(ctx sdk.Context) valuefeetypes.Params
}

type RefundUnusedGasDecorator struct {
	accountKeeper  AccountKeeper
	bankKeeper     BankKeeper
	valueFeeKeeper ValueFeeKeeper
}

func NewRefundUnusedGasDecorator(ak AccountKeeper, bk BankKeeper, vk ValueFeeKeeper) RefundUnusedGasDecorator {
	return RefundUnusedGasDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		valueFeeKeeper: vk,
	}
}

func (rd RefundUnusedGasDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {
	newCtx, err := next(ctx, tx, simulate, success)
	if err != nil {
		return newCtx, err
	}

	if simulate || newCtx.IsCheckTx() || !success || rd.valueFeeKeeper == nil {
		return newCtx, nil
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return newCtx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	if helper.IsOracleTx(feeTx.GetMsgs()) {
		return newCtx, nil
	}

	params := rd.valueFeeKeeper.GetParams(newCtx)
	if params.PolicyVersion < 23 || !params.RefundUnusedGas || params.NormalGasPrices.IsZero() {
		return newCtx, nil
	}
	if err := params.Validate(); err != nil {
		return newCtx, err
	}
	if rd.isPureValueSend(newCtx, feeTx.GetMsgs(), params) {
		return newCtx, nil
	}

	gasLimit := feeTx.GetGas()
	gasConsumed := newCtx.GasMeter().GasConsumed()
	if gasLimit == 0 || gasConsumed >= gasLimit {
		return newCtx, nil
	}

	gasLimitFees := gasPricesToFees(params.NormalGasPrices, gasLimit)
	gasUsedFees := gasPricesToFees(params.NormalGasPrices, gasConsumed)
	refund := refundableGasFees(feeTx.GetFee(), gasLimitFees, gasUsedFees)
	if refund.IsZero() {
		return newCtx, nil
	}

	feePayer, err := feeDeductedFrom(feeTx)
	if err != nil {
		return newCtx, err
	}
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

func gasPricesToFees(gasPrices sdk.DecCoins, gas uint64) sdk.Coins {
	if gasPrices.IsZero() {
		return sdk.NewCoins()
	}

	requiredFees := make(sdk.Coins, len(gasPrices))
	glDec := sdkmath.LegacyNewDecFromInt(sdkmath.NewIntFromUint64(gas))
	for i, gp := range gasPrices {
		fee := gp.Amount.Mul(glDec)
		requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}
	return requiredFees.Sort()
}

func refundableGasFees(paidFees, gasLimitFees, gasUsedFees sdk.Coins) sdk.Coins {
	for _, gasLimitFee := range gasLimitFees {
		paidAmount := paidFees.AmountOf(gasLimitFee.Denom)
		if paidAmount.LT(gasLimitFee.Amount) {
			continue
		}
		usedAmount := gasUsedFees.AmountOf(gasLimitFee.Denom)
		if usedAmount.GTE(gasLimitFee.Amount) {
			return sdk.NewCoins()
		}
		return sdk.NewCoins(sdk.NewCoin(gasLimitFee.Denom, gasLimitFee.Amount.Sub(usedAmount)))
	}
	return sdk.NewCoins()
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

func (rd RefundUnusedGasDecorator) isPureValueSend(ctx sdk.Context, msgs []sdk.Msg, params valuefeetypes.Params) bool {
	if !params.Enabled || !params.ApplyToMsgSend || len(msgs) == 0 {
		return false
	}

	for _, msg := range msgs {
		sendMsg, ok := msg.(*banktypes.MsgSend)
		if !ok {
			return false
		}
		fromAddr, err := sdk.AccAddressFromBech32(sendMsg.FromAddress)
		if err != nil {
			return false
		}
		toAddr, err := sdk.AccAddressFromBech32(sendMsg.ToAddress)
		if err != nil {
			return false
		}
		if rd.isModuleAccount(ctx, fromAddr) || rd.isModuleAccount(ctx, toAddr) {
			return false
		}

		msgHasValueFee := false
		for _, coin := range sendMsg.Amount {
			if hasDenomValueRate(params, coin.Denom) || params.ChargeUnknownDenomMinFee {
				msgHasValueFee = true
				continue
			}
			return false
		}
		if !msgHasValueFee {
			return false
		}
	}
	return true
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

func hasDenomValueRate(params valuefeetypes.Params, denom string) bool {
	for _, rate := range params.DenomValueRates {
		if rate.Denom == denom {
			return true
		}
	}
	return false
}
