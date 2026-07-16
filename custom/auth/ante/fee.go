package ante

import (
	"bytes"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/app/helper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

type FeeDecorator struct {
	accountKeeper  ante.AccountKeeper
	bankKeeper     BankKeeper
	feegrantKeeper ante.FeegrantKeeper
	treasuryKeeper TreasuryKeeper
	distrKeeper    DistrKeeper
}

func NewFeeDecorator(ak ante.AccountKeeper, bk BankKeeper, fk ante.FeegrantKeeper, tk TreasuryKeeper, dk DistrKeeper) FeeDecorator {
	return FeeDecorator{
		accountKeeper:  ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,
		treasuryKeeper: tk,
		distrKeeper:    dk,
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

	if !isOracleTx {
		minGasPrices := ctx.MinGasPrices()
		if !minGasPrices.IsZero() {
			requiredFees := make(sdk.Coins, len(minGasPrices))
			glDec := sdkmath.LegacyNewDec(int64(gas))

			for i, gp := range minGasPrices {
				fee := gp.Amount.Mul(glDec)
				requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
			}

			if !feeCoins.IsAnyGTE(requiredFees) {
				return 0, false, false, errorsmod.Wrapf(
					sdkerrors.ErrInsufficientFee,
					"insufficient fee; got: %s required: %s",
					feeCoins, requiredFees,
				)
			}
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
