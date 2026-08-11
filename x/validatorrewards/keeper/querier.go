package keeper

import (
	"context"

	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) types.QueryServer {
	return &querier{Keeper: keeper}
}

var _ types.QueryServer = querier{}

func (q querier) RewardDenom(c context.Context, req *types.QueryRewardDenomRequest) (*types.QueryRewardDenomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	record, found := q.GetRewardDenom(ctx, req.Denom)
	if !found {
		return nil, status.Error(codes.NotFound, "reward denom not found")
	}
	return &types.QueryRewardDenomResponse{RewardDenom: &record}, nil
}

func (q querier) RewardDenoms(c context.Context, req *types.QueryRewardDenomsRequest) (*types.QueryRewardDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	var records []*types.RewardDenom
	q.IterateRewardDenoms(ctx, func(record types.RewardDenom) bool {
		recordCopy := record
		records = append(records, &recordCopy)
		return false
	})
	start, end := paginateBounds(req.Pagination, len(records))
	return &types.QueryRewardDenomsResponse{
		RewardDenoms: records[start:end],
		Pagination:   &query.PageResponse{Total: uint64(len(records))},
	}, nil
}

func (q querier) Campaign(c context.Context, req *types.QueryCampaignRequest) (*types.QueryCampaignResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	campaign, found := q.GetCampaign(ctx, req.CampaignId)
	if !found {
		return nil, status.Error(codes.NotFound, "campaign not found")
	}
	return &types.QueryCampaignResponse{Campaign: &campaign}, nil
}

func (q querier) Campaigns(c context.Context, req *types.QueryCampaignsRequest) (*types.QueryCampaignsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	var campaigns []*types.Campaign
	q.IterateCampaigns(ctx, func(campaign types.Campaign) bool {
		campaignCopy := campaign
		campaigns = append(campaigns, &campaignCopy)
		return false
	})
	start, end := paginateBounds(req.Pagination, len(campaigns))
	return &types.QueryCampaignsResponse{
		Campaigns:  campaigns[start:end],
		Pagination: &query.PageResponse{Total: uint64(len(campaigns))},
	}, nil
}

func (q querier) ValidatorCampaigns(c context.Context, req *types.QueryValidatorCampaignsRequest) (*types.QueryValidatorCampaignsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	if _, err := q.StakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var campaigns []*types.Campaign
	q.IterateValidatorCampaigns(ctx, req.ValidatorAddress, func(campaign types.Campaign) bool {
		campaignCopy := campaign
		campaigns = append(campaigns, &campaignCopy)
		return false
	})
	start, end := paginateBounds(req.Pagination, len(campaigns))
	return &types.QueryValidatorCampaignsResponse{
		Campaigns:  campaigns[start:end],
		Pagination: &query.PageResponse{Total: uint64(len(campaigns))},
	}, nil
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
