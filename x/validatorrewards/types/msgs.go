package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgRegisterRewardDenom{}
	_ sdk.Msg = &MsgCreateCampaign{}
	_ sdk.Msg = &MsgReleaseCampaign{}
	_ sdk.Msg = &MsgCancelCampaign{}
)

const (
	TypeMsgRegisterRewardDenom = "register_reward_denom"
	TypeMsgCreateCampaign      = "create_campaign"
	TypeMsgReleaseCampaign     = "release_campaign"
	TypeMsgCancelCampaign      = "cancel_campaign"
)

func NewMsgRegisterRewardDenom(authority sdk.AccAddress, denom string, admin sdk.AccAddress) *MsgRegisterRewardDenom {
	return &MsgRegisterRewardDenom{Authority: authority.String(), Denom: denom, Admin: admin.String()}
}

func (msg MsgRegisterRewardDenom) Route() string { return RouterKey }
func (msg MsgRegisterRewardDenom) Type() string  { return TypeMsgRegisterRewardDenom }
func (msg MsgRegisterRewardDenom) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}
func (msg MsgRegisterRewardDenom) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
func (msg MsgRegisterRewardDenom) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Admin); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}
	return ValidateRewardDenom(msg.Denom)
}

func NewMsgCreateCampaign(admin sdk.AccAddress, denom, title, description string, allocations []ValidatorAllocation, startHeight, endHeight int64, delegatorsOnly bool) *MsgCreateCampaign {
	return &MsgCreateCampaign{
		Admin:          admin.String(),
		Denom:          denom,
		Title:          title,
		Description:    description,
		Allocations:    allocations,
		StartHeight:    startHeight,
		EndHeight:      endHeight,
		DelegatorsOnly: delegatorsOnly,
	}
}

func (msg MsgCreateCampaign) Route() string { return RouterKey }
func (msg MsgCreateCampaign) Type() string  { return TypeMsgCreateCampaign }
func (msg MsgCreateCampaign) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}
func (msg MsgCreateCampaign) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
func (msg MsgCreateCampaign) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Admin); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}
	if err := ValidateRewardDenom(msg.Denom); err != nil {
		return err
	}
	if len(msg.Allocations) == 0 {
		return errorsmod.Wrap(ErrInvalidCampaign, "at least one validator allocation is required")
	}
	if msg.EndHeight != 0 && msg.EndHeight <= msg.StartHeight {
		return errorsmod.Wrap(ErrInvalidCampaign, "end_height must be greater than start_height")
	}
	seen := map[string]bool{}
	for _, allocation := range msg.Allocations {
		if err := ValidateAllocation(msg.Denom, allocation); err != nil {
			return err
		}
		if seen[allocation.ValidatorAddress] {
			return errorsmod.Wrapf(ErrInvalidCampaign, "duplicate validator %s", allocation.ValidatorAddress)
		}
		seen[allocation.ValidatorAddress] = true
	}
	return nil
}

func NewMsgReleaseCampaign(releaser sdk.AccAddress, campaignID uint64) *MsgReleaseCampaign {
	return &MsgReleaseCampaign{Releaser: releaser.String(), CampaignId: campaignID}
}

func (msg MsgReleaseCampaign) Route() string { return RouterKey }
func (msg MsgReleaseCampaign) Type() string  { return TypeMsgReleaseCampaign }
func (msg MsgReleaseCampaign) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}
func (msg MsgReleaseCampaign) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Releaser)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
func (msg MsgReleaseCampaign) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Releaser); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid releaser address (%s)", err)
	}
	if msg.CampaignId == 0 {
		return errorsmod.Wrap(ErrInvalidCampaign, "campaign_id must be positive")
	}
	return nil
}

func NewMsgCancelCampaign(admin sdk.AccAddress, campaignID uint64) *MsgCancelCampaign {
	return &MsgCancelCampaign{Admin: admin.String(), CampaignId: campaignID}
}

func (msg MsgCancelCampaign) Route() string { return RouterKey }
func (msg MsgCancelCampaign) Type() string  { return TypeMsgCancelCampaign }
func (msg MsgCancelCampaign) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}
func (msg MsgCancelCampaign) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
func (msg MsgCancelCampaign) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Admin); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
	}
	if msg.CampaignId == 0 {
		return errorsmod.Wrap(ErrInvalidCampaign, "campaign_id must be positive")
	}
	return nil
}

func ValidateRewardDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return errorsmod.Wrap(ErrInvalidDenom, err.Error())
	}
	return nil
}

func ValidateAllocation(denom string, allocation ValidatorAllocation) error {
	if allocation.ValidatorAddress == "" {
		return errorsmod.Wrap(ErrInvalidCampaign, "validator address is required")
	}
	if allocation.Amount.Denom != denom {
		return errorsmod.Wrapf(ErrInvalidCampaign, "allocation denom %s does not match campaign denom %s", allocation.Amount.Denom, denom)
	}
	if !allocation.Amount.IsValid() || !allocation.Amount.Amount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidCampaign, "allocation amount must be positive")
	}
	released, ok := sdkmath.NewIntFromString(defaultZero(allocation.Released))
	if !ok || released.IsNegative() || released.GT(allocation.Amount.Amount) {
		return errorsmod.Wrap(ErrInvalidCampaign, "released amount is invalid")
	}
	return nil
}

func defaultZero(value string) string {
	if value == "" {
		return "0"
	}
	return value
}
