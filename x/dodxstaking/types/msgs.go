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
	_ sdk.Msg = &MsgDepositRewards{}
	_ sdk.Msg = &MsgClaimRewards{}
)

const (
	TypeMsgStake          = "stake"
	TypeMsgUnstake        = "unstake"
	TypeMsgDepositRewards = "deposit_rewards"
	TypeMsgClaimRewards   = "claim_rewards"
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

// NewMsgDepositRewards creates a MsgDepositRewards instance.
func NewMsgDepositRewards(depositor sdk.AccAddress, amount sdk.Coins) *MsgDepositRewards {
	return &MsgDepositRewards{
		Depositor: depositor.String(),
		Amount:    amount,
	}
}

func (msg MsgDepositRewards) Route() string { return RouterKey }

func (msg MsgDepositRewards) Type() string { return TypeMsgDepositRewards }

func (msg MsgDepositRewards) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgDepositRewards) GetSigners() []sdk.AccAddress {
	depositor, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{depositor}
}

func (msg MsgDepositRewards) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid depositor address (%s)", err)
	}

	if !msg.Amount.IsValid() || !msg.Amount.IsAllPositive() {
		return errorsmod.Wrap(ErrInvalidRewardAmount, msg.Amount.String())
	}

	for _, coin := range msg.Amount {
		if err := validateRewardDenom(coin.Denom); err != nil {
			return err
		}
	}

	return nil
}

// NewMsgClaimRewards creates a MsgClaimRewards instance.
func NewMsgClaimRewards(claimer sdk.AccAddress, denoms []string) *MsgClaimRewards {
	return &MsgClaimRewards{
		Claimer: claimer.String(),
		Denoms:  denoms,
	}
}

func (msg MsgClaimRewards) Route() string { return RouterKey }

func (msg MsgClaimRewards) Type() string { return TypeMsgClaimRewards }

func (msg MsgClaimRewards) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

func (msg MsgClaimRewards) GetSigners() []sdk.AccAddress {
	claimer, err := sdk.AccAddressFromBech32(msg.Claimer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{claimer}
}

func (msg MsgClaimRewards) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Claimer); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid claimer address (%s)", err)
	}

	seen := map[string]bool{}
	for _, denom := range msg.Denoms {
		if seen[denom] {
			return errorsmod.Wrapf(ErrInvalidRewardDenom, "duplicate denom %s", denom)
		}
		seen[denom] = true
		if err := validateRewardDenom(denom); err != nil {
			return err
		}
	}

	return nil
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
