package mfa

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/x/mfa/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/mfa/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, gs types.GenesisState) {
	if err := types.ValidateGenesis(gs); err != nil {
		panic(err)
	}
	for _, policy := range gs.Policies {
		if err := k.SetPolicy(ctx, policy); err != nil {
			panic(err)
		}
	}
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) types.GenesisState {
	policies := make([]types.Policy, 0)
	store := ctx.KVStore(k.StoreKey())
	iterator := storetypes.KVStorePrefixIterator(store, types.PolicyKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		account := string(iterator.Key()[len(types.PolicyKeyPrefix):])
		addr, err := sdk.AccAddressFromBech32(account)
		if err != nil {
			continue
		}
		if policy, found := k.GetPolicy(ctx, addr); found {
			policies = append(policies, policy)
		}
	}
	return types.GenesisState{Policies: policies}
}
