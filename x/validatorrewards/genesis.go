package validatorrewards

import (
	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, data *types.GenesisState) {
	if data.NextCampaignId == 0 {
		data.NextCampaignId = 1
	}
	k.SetNextCampaignID(ctx, data.NextCampaignId)
	for _, record := range data.RewardDenoms {
		if record != nil {
			k.SetRewardDenom(ctx, *record)
		}
	}
	for _, campaign := range data.Campaigns {
		if campaign != nil {
			k.SetCampaign(ctx, *campaign)
		}
	}
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	gs := &types.GenesisState{NextCampaignId: k.GetNextCampaignID(ctx)}
	k.IterateRewardDenoms(ctx, func(record types.RewardDenom) bool {
		recordCopy := record
		gs.RewardDenoms = append(gs.RewardDenoms, &recordCopy)
		return false
	})
	k.IterateCampaigns(ctx, func(campaign types.Campaign) bool {
		campaignCopy := campaign
		gs.Campaigns = append(gs.Campaigns, &campaignCopy)
		return false
	})
	return gs
}
