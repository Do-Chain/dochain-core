package keeper

import (
	"context"
	"fmt"

	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) RegisterRewardDenom(goCtx context.Context, msg *types.MsgRegisterRewardDenom) (*types.MsgRegisterRewardDenomResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if msg.Authority != k.Authority() {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.Authority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.ValidateBankDenom(ctx, msg.Denom); err != nil {
		return nil, err
	}
	k.SetRewardDenom(ctx, types.RewardDenom{Denom: msg.Denom, Admin: msg.Admin})
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRegisterRewardDenom,
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyAdmin, msg.Admin),
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
		),
	)
	return &types.MsgRegisterRewardDenomResponse{}, nil
}

func (k msgServer) CreateCampaign(goCtx context.Context, msg *types.MsgCreateCampaign) (*types.MsgCreateCampaignResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	record, found := k.GetRewardDenom(ctx, msg.Denom)
	if !found {
		return nil, types.ErrRewardDenomMissing.Wrap(msg.Denom)
	}
	if record.Admin != msg.Admin {
		return nil, types.ErrUnauthorizedAdmin.Wrapf("expected %s, got %s", record.Admin, msg.Admin)
	}
	if err := k.ValidateBankDenom(ctx, msg.Denom); err != nil {
		return nil, err
	}

	total := sdk.NewCoins()
	for _, allocation := range msg.Allocations {
		valAddr, err := k.StakingKeeper.ValidatorAddressCodec().StringToBytes(allocation.ValidatorAddress)
		if err != nil {
			return nil, err
		}
		validator, err := k.StakingKeeper.Validator(ctx, valAddr)
		if err != nil {
			return nil, err
		}
		if validator == nil {
			return nil, fmt.Errorf("%w: %s", types.ErrInvalidCampaign, allocation.ValidatorAddress)
		}
		total = total.Add(allocation.Amount)
	}

	admin, err := sdk.AccAddressFromBech32(msg.Admin)
	if err != nil {
		return nil, err
	}
	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, admin, types.ModuleName, total); err != nil {
		return nil, err
	}

	id := k.GetNextCampaignID(ctx)
	campaign := types.Campaign{
		Id:             id,
		Admin:          msg.Admin,
		Denom:          msg.Denom,
		Title:          msg.Title,
		Description:    msg.Description,
		Allocations:    msg.Allocations,
		StartHeight:    msg.StartHeight,
		EndHeight:      msg.EndHeight,
		DelegatorsOnly: msg.DelegatorsOnly,
	}
	for i := range campaign.Allocations {
		campaign.Allocations[i].Released = defaultZero(campaign.Allocations[i].Released)
	}
	k.SetCampaign(ctx, campaign)
	k.SetNextCampaignID(ctx, id+1)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCreateCampaign,
			sdk.NewAttribute(types.AttributeKeyCampaign, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(types.AttributeKeyAdmin, msg.Admin),
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
			sdk.NewAttribute(types.AttributeKeyAmount, total.String()),
		),
	)

	return &types.MsgCreateCampaignResponse{CampaignId: id}, nil
}

func (k msgServer) ReleaseCampaign(goCtx context.Context, msg *types.MsgReleaseCampaign) (*types.MsgReleaseCampaignResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	campaign, found := k.GetCampaign(ctx, msg.CampaignId)
	if !found {
		return nil, types.ErrCampaignNotFound.Wrapf("%d", msg.CampaignId)
	}
	released, err := k.Keeper.ReleaseCampaign(ctx, campaign)
	if err != nil {
		return nil, err
	}
	return &types.MsgReleaseCampaignResponse{Released: released}, nil
}

func (k msgServer) CancelCampaign(goCtx context.Context, msg *types.MsgCancelCampaign) (*types.MsgCancelCampaignResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	campaign, found := k.GetCampaign(ctx, msg.CampaignId)
	if !found {
		return nil, types.ErrCampaignNotFound.Wrapf("%d", msg.CampaignId)
	}
	if campaign.Admin != msg.Admin {
		return nil, types.ErrUnauthorizedAdmin.Wrapf("expected %s, got %s", campaign.Admin, msg.Admin)
	}
	refunded, err := k.Keeper.CancelCampaign(ctx, campaign)
	if err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCancelCampaign,
			sdk.NewAttribute(types.AttributeKeyCampaign, fmt.Sprintf("%d", msg.CampaignId)),
			sdk.NewAttribute(types.AttributeKeyAmount, refunded.String()),
		),
	)
	return &types.MsgCancelCampaignResponse{Refunded: refunded}, nil
}
