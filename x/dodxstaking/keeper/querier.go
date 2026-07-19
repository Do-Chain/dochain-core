package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
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

func (q querier) Stakes(c context.Context, req *types.QueryStakesRequest) (*types.QueryStakesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	var stakes []types.StakeRecord
	q.IterateStakes(ctx, func(addr sdk.AccAddress, amount sdkmath.Int) bool {
		stakes = append(stakes, types.StakeRecord{
			Address: addr.String(),
			Amount:  sdk.NewCoin(core.MicroDODxDenom, amount),
		})
		return false
	})

	start, end := paginateBounds(req.Pagination, len(stakes))
	return &types.QueryStakesResponse{
		Stakes: stakes[start:end],
		Pagination: &query.PageResponse{
			Total: uint64(len(stakes)),
		},
	}, nil
}

func (q querier) PendingRewards(c context.Context, req *types.QueryPendingRewardsRequest) (*types.QueryPendingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryPendingRewardsResponse{Rewards: q.Keeper.PendingRewards(ctx, addr)}, nil
}

func (q querier) RewardPool(c context.Context, _ *types.QueryRewardPoolRequest) (*types.QueryRewardPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryRewardPoolResponse{Rewards: q.Keeper.RewardPool(ctx)}, nil
}

func paginateBounds(req *query.PageRequest, total int) (int, int) {
	if req == nil {
		return 0, total
	}
	start := int(req.Offset)
	if start > total {
		start = total
	}
	limit := int(req.Limit)
	if limit <= 0 {
		return start, total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return start, end
}
