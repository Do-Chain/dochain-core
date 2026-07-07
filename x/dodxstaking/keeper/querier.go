package keeper

import (
	"context"

	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the DODx staking QueryServer.
func NewQuerier(keeper Keeper) types.QueryServer {
	return &querier{Keeper: keeper}
}

var _ types.QueryServer = querier{}

func (q querier) Stake(c context.Context, req *types.QueryStakeRequest) (*types.QueryStakeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryStakeResponse{Amount: q.GetStake(ctx, addr)}, nil
}

func (q querier) TotalStaked(c context.Context, _ *types.QueryTotalStakedRequest) (*types.QueryTotalStakedResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryTotalStakedResponse{Amount: q.GetTotalStaked(ctx)}, nil
}
