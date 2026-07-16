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

	k.AddStake(ctx, staker, msg.Amount.Amount)
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

	if err := k.RemoveStake(ctx, staker, msg.Amount.Amount); err != nil {
		return nil, err
	}

	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, staker, sdk.NewCoins(msg.Amount)); err != nil {
		return nil, err
	}

	emitStakeEvent(ctx, types.EventTypeUnstake, msg.Staker, msg.Amount)

	return &types.MsgUnstakeResponse{}, nil
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
