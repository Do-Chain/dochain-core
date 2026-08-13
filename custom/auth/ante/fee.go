package ante

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/app/helper"
	core "github.com/Daviddochain/dochain-core/v4/types"
	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type FeeDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     BankKeeper
	feegrantKeeper ante.FeegrantKeeper
	treasuryKeeper TreasuryKeeper
	distrKeeper    DistrKeeper
	valueFeeKeeper ValueFeeKeeper
}

func NewFeeDecorator(ak ante.AccountKeeper, bk BankKeeper, fk ante.FeegrantKeeper, tk TreasuryKeeper, dk DistrKeeper, vk ValueFeeKeeper) FeeDecorator {
	return FeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		treasuryKeeper: tk,
		distrKeeper:    dk,
		valueFeeKeeper: vk,
	}
}

func (fd FeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidGasLimit, "must provide positive gas")
	}

	var priority int64
	var err error

	taxes, nonTaxableTaxes := FilterMsgAndComputeTax(ctx, feeTx.GetMsgs()...)

	if !simulate {
		priority, _, _, err = fd.checkTxFee(ctx, tx, taxes, nonTaxableTaxes)
		if err != nil {
			return ctx, err
		}
	}

	newCtx, err := fd.checkDeductFee(ctx, feeTx, taxes, nonTaxableTaxes, simulate)
	if err != nil {
		return newCtx, err
	}

	newCtx = newCtx.WithPriority(priority)
	return next(newCtx, tx, simulate)
}

func (fd FeeDecorator) checkDeductFee(ctx sdk.Context, feeTx sdk.FeeTx, taxes sdk.Coins, nonTaxableTaxes sdk.Coins, simulate bool) (sdk.Context, error) {
	if addr := fd.accountKeeper.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	fee := feeTx.GetFee()
	if helper.IsOracleTx(feeTx.GetMsgs()) {
		fee = sdk.Coins{}
	}
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	if len(feePayer) == 0 {
		if sigTx, ok := feeTx.(authsigning.SigVerifiableTx); ok {
			signers, err := sigTx.GetSigners()
			if err != nil {
				return ctx, fmt.Errorf("fee payer address not found and cannot get signers: %v", err)
			}
			if len(signers) == 0 {
				return ctx, fmt.Errorf("fee payer address not found and no signers available")
			}
			feePayer = signers[0]
		} else {
			return ctx, fmt.Errorf("fee payer address not found and cannot cast to SigVerifiableTx")
		}
	}

	deductFeesFrom := feePayer

	if feeGranter != nil {
		if fd.feegrantKeeper == nil {
			return ctx, sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !bytes.Equal(feeGranter, feePayer) {
			err := fd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, feeTx.GetMsgs())
			if err != nil {
				return ctx, errorsmod.Wrapf(err, "%s does not not allow to pay fees for %s", feeGranter, feePayer)
			}
		}
		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := fd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	feesToDeduct := fee
	if simulate && fee.IsZero() {
		feesToDeduct = taxes
	}

	if !feesToDeduct.IsZero() {
		if err := DeductFees(fd.bankKeeper, ctx, deductFeesFromAcc, feesToDeduct); err != nil {
			return ctx, err
		}
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, sdk.AccAddress(deductFeesFrom).String()),
		),
	}
	ctx.EventManager().EmitEvents(events)

	return ctx, nil
}

func DeductFees(bankKeeper BankKeeper, ctx sdk.Context, acc types.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(sdk.WrapSDKContext(ctx), acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, "%s", err.Error())
	}

	return nil
}

func (fd FeeDecorator) checkTxFee(ctx sdk.Context, tx sdk.Tx, taxes sdk.Coins, nonTaxableTaxes sdk.Coins) (int64, bool, bool, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return 0, false, false, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()
	isOracleTx := helper.IsOracleTx(msgs)
	valueFee, hasValueFee, pureValueSend, err := fd.computeValueSendFee(ctx, msgs)
	if err != nil {
		return 0, false, false, err
	}

	if !isOracleTx {
		requiredGasFees := requiredGasFees(ctx, gas)
		if !pureValueSend {
			if err := checkRequiredGasAndValueFees(feeCoins, requiredGasFees, valueFee, hasValueFee); err != nil {
				return 0, false, false, err
			}
		} else if hasValueFee && !feeCoins.IsAllGTE(sdk.NewCoins(valueFee)) {
			return 0, false, false, errorsmod.Wrapf(
				sdkerrors.ErrInsufficientFee,
				"insufficient fee; got: %s required: %s",
				feeCoins, sdk.NewCoins(valueFee),
			)
		}
	}

	var priority int64
	for _, c := range feeCoins {
		gasPrice := c.Amount.QuoRaw(int64(gas))
		if gasPrice.IsInt64() && gasPrice.Int64() > priority {
			priority = gasPrice.Int64()
		}
	}

	return priority, false, false, nil
}

func requiredGasFees(ctx sdk.Context, gas uint64) sdk.Coins {
	minGasPrices := ctx.MinGasPrices()
	if minGasPrices.IsZero() {
		return sdk.NewCoins()
	}

	requiredFees := make(sdk.Coins, len(minGasPrices))
	glDec := sdkmath.LegacyNewDec(int64(gas))
	for i, gp := range minGasPrices {
		fee := gp.Amount.Mul(glDec)
		requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}
	return requiredFees.Sort()
}

func checkRequiredGasAndValueFees(feeCoins, gasFees sdk.Coins, valueFee sdk.Coin, hasValueFee bool) error {
	if gasFees.IsZero() {
		if !hasValueFee || feeCoins.IsAllGTE(sdk.NewCoins(valueFee)) {
			return nil
		}
		return errorsmod.Wrapf(
			sdkerrors.ErrInsufficientFee,
			"insufficient fee; got: %s required: %s",
			feeCoins, sdk.NewCoins(valueFee),
		)
	}

	for _, gasFee := range gasFees {
		required := sdk.NewCoins(gasFee)
		if hasValueFee {
			required = required.Add(valueFee)
		}
		if feeCoins.IsAllGTE(required) {
			return nil
		}
	}

	required := gasFees
	if hasValueFee {
		required = required.Add(valueFee)
	}
	return errorsmod.Wrapf(
		sdkerrors.ErrInsufficientFee,
		"insufficient fee; got: %s required: %s",
		feeCoins, required,
	)
}

func (fd FeeDecorator) computeValueSendFee(ctx sdk.Context, msgs []sdk.Msg) (sdk.Coin, bool, bool, error) {
	zero := sdk.NewCoin(core.MicroDoDenom, sdkmath.ZeroInt())
	if fd.valueFeeKeeper == nil {
		return zero, false, false, nil
	}

	params := fd.valueFeeKeeper.GetParams(ctx)
	if !params.Enabled || !params.ApplyToMsgSend || params.PolicyVersion == 0 {
		return zero, false, false, nil
	}
	if err := params.Validate(); err != nil {
		return zero, false, false, err
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
		if fd.isModuleAccount(ctx, fromAddr) || fd.isModuleAccount(ctx, toAddr) {
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

func denomValueInFeeDenom(coin sdk.Coin, params valuefeetypes.Params) (sdkmath.Int, bool) {
	for _, rate := range params.DenomValueRates {
		if rate.Denom == coin.Denom {
			return coin.Amount.Mul(rate.UdoPerBaseUnit), true
		}
	}
	return sdkmath.ZeroInt(), false
}

func (fd FeeDecorator) isModuleAccount(ctx sdk.Context, addr sdk.AccAddress) bool {
	acc := fd.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return false
	}
	_, ok := acc.(types.ModuleAccountI)
	return ok
}
