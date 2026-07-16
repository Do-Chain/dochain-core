package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgStake{}
	_ sdk.Msg = &MsgUnstake{}
)

const (
	TypeMsgStake   = "stake"
	TypeMsgUnstake = "unstake"
)

// NewMsgStake creates a MsgStake instance.
func NewMsgStake(staker sdk.AccAddress, amount sdk.Coin) *MsgStake {
	return &MsgStake{
		Staker: staker.String(),
		Amount: amount,
	}
}

func (msg MsgStake) Route() string { return RouterKey }

func (msg MsgStake) Type() string { return TypeMsgStake }

func (msg MsgStake) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgStake) GetSigners() []sdk.AccAddress {
	staker, err := sdk.AccAddressFromBech32(msg.Staker)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{staker}
}

func (msg MsgStake) ValidateBasic() error {
	return validateStakeMsg(msg.Staker, msg.Amount)
}

// NewMsgUnstake creates a MsgUnstake instance.
func NewMsgUnstake(staker sdk.AccAddress, amount sdk.Coin) *MsgUnstake {
	return &MsgUnstake{
		Staker: staker.String(),
		Amount: amount,
	}
}

func (msg MsgUnstake) Route() string { return RouterKey }

func (msg MsgUnstake) Type() string { return TypeMsgUnstake }

func (msg MsgUnstake) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgUnstake) GetSigners() []sdk.AccAddress {
	staker, err := sdk.AccAddressFromBech32(msg.Staker)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{staker}
}

func (msg MsgUnstake) ValidateBasic() error {
	return validateStakeMsg(msg.Staker, msg.Amount)
}

func validateStakeMsg(staker string, amount sdk.Coin) error {
	if _, err := sdk.AccAddressFromBech32(staker); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid staker address (%s)", err)
	}

	if amount.Denom != core.MicroDODxDenom {
		return errorsmod.Wrapf(ErrInvalidStakeDenom, "got %s", amount.Denom)
	}

	if !amount.IsValid() || amount.Amount.LTE(sdkmath.ZeroInt()) || amount.Amount.BigInt().BitLen() > 100 {
		return errorsmod.Wrap(ErrInvalidAmount, amount.String())
	}

	return nil
}
