package types

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesisState() *GenesisState {
	return &GenesisState{NextCampaignId: 1}
}

func ValidateGenesis(data *GenesisState) error {
	if data.NextCampaignId == 0 {
		return fmt.Errorf("next_campaign_id must be positive")
	}

	seenDenoms := map[string]bool{}
	for _, record := range data.RewardDenoms {
		if record == nil {
			return fmt.Errorf("nil reward denom")
		}
		if err := ValidateRewardDenom(record.Denom); err != nil {
			return err
		}
		if _, err := sdk.AccAddressFromBech32(record.Admin); err != nil {
			return err
		}
		if seenDenoms[record.Denom] {
			return fmt.Errorf("duplicate reward denom %s", record.Denom)
		}
		seenDenoms[record.Denom] = true
	}

	seenCampaigns := map[uint64]bool{}
	for _, campaign := range data.Campaigns {
		if campaign == nil {
			return fmt.Errorf("nil campaign")
		}
		if campaign.Id == 0 {
			return fmt.Errorf("campaign id must be positive")
		}
		if seenCampaigns[campaign.Id] {
			return fmt.Errorf("duplicate campaign id %d", campaign.Id)
		}
		seenCampaigns[campaign.Id] = true
		if _, err := sdk.AccAddressFromBech32(campaign.Admin); err != nil {
			return err
		}
		if err := ValidateRewardDenom(campaign.Denom); err != nil {
			return err
		}
		if campaign.EndHeight != 0 && campaign.EndHeight <= campaign.StartHeight {
			return fmt.Errorf("campaign %d end_height must be greater than start_height", campaign.Id)
		}
		for _, allocation := range campaign.Allocations {
			if err := ValidateAllocation(campaign.Denom, allocation); err != nil {
				return err
			}
		}
	}

	return nil
}

func GetGenesisStateFromAppState(_ codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState
	if appState[ModuleName] != nil {
		if err := json.Unmarshal(appState[ModuleName], &genesisState); err != nil {
			panic(err)
		}
	}
	if genesisState.NextCampaignId == 0 {
		genesisState.NextCampaignId = 1
	}
	return &genesisState
}
