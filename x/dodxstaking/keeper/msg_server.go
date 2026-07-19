package keeper

import (
	"context"

	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the DODx staking MsgServer.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Stake(goCtx context.Context, msg *types.MsgStake) (*types.MsgStakeResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	staker, err := sdk.AccAddressFromBech32(msg.Staker)
	if err != nil {
		return nil, err
	}

	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, staker, types.ModuleName, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}

	k.SettleRewards(ctx, staker)
	k.AddStake(ctx, staker, msg.Amount.Amount)
	k.ResetRewardDebts(ctx, staker)
	emitStakeEvent(ctx, types.EventTypeStake, msg.Staker, msg.Amount)

	return &types.MsgStakeResponse{}, nil
}

func (k msgServer) Unstake(goCtx context.Context, msg *types.MsgUnstake) (*types.MsgUnstakeResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	staker, err := sdk.AccAddressFromBech32(msg.Staker)
	if err != nil {
		return nil, err
	}

	k.SettleRewards(ctx, staker)
	if err := k.RemoveStake(ctx, staker, msg.Amount.Amount); err != nil {
		return nil, err
	}
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, staker, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}
	k.ResetRewardDebts(ctx, staker)

	emitStakeEvent(ctx, types.EventTypeUnstake, msg.Staker, msg.Amount)

	return &types.MsgUnstakeResponse{}, nil
}

func (k msgServer) DepositRewards(goCtx context.Context, msg *types.MsgDepositRewards) (*types.MsgDepositRewardsResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	depositor, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, err
	}

	if err := k.BankKeeper.SendCoinsFromAccountToModule(ctx, depositor, types.ModuleName, msg.Amount); err != nil {
		return nil, err
	}

	for _, coin := range msg.Amount {
		k.CreditRewards(ctx, coin)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDepositRewards,
			sdk.NewAttribute(types.AttributeKeyDepositor, msg.Depositor),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		),
	)

	return &types.MsgDepositRewardsResponse{}, nil
}

func (k msgServer) ClaimRewards(goCtx context.Context, msg *types.MsgClaimRewards) (*types.MsgClaimRewardsResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	claimer, err := sdk.AccAddressFromBech32(msg.Claimer)
	if err != nil {
		return nil, err
	}

	k.SyncRewardBalances(ctx)
	claimed, err := k.ClaimPendingRewards(ctx, claimer, msg.Denoms)
	if err != nil {
		return nil, err
	}

	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimer, claimed); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeClaimRewards,
			sdk.NewAttribute(types.AttributeKeyClaimer, msg.Claimer),
			sdk.NewAttribute(types.AttributeKeyAmount, claimed.String()),
		),
	)

	return &types.MsgClaimRewardsResponse{Amount: claimed}, nil
}

func emitStakeEvent(ctx sdk.Context, eventType, staker string, amount sdk.Coin) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttributeKeyStaker, staker),
			sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})
}
