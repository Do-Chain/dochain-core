package keeper

import (
	"encoding/json"

	"github.com/Daviddochain/dochain-core/v4/x/mfa/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetPolicy(ctx sdk.Context, policy types.Policy) error {
	if err := policy.ValidateBasic(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	store.Set(types.PolicyKey(policy.Account), bz)
	return nil
}

func (k Keeper) GetPolicy(ctx sdk.Context, account sdk.AccAddress) (types.Policy, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PolicyKey(account.String()))
	if bz == nil {
		return types.Policy{}, false
	}
	var policy types.Policy
	if err := json.Unmarshal(bz, &policy); err != nil {
		k.Logger(ctx).Error("invalid mfa policy state", "account", account.String(), "error", err)
		return types.Policy{Account: account.String(), Enabled: true}, true
	}
	return policy, policy.Enabled
}

func (k Keeper) DeletePolicy(ctx sdk.Context, account sdk.AccAddress) {
	ctx.KVStore(k.storeKey).Delete(types.PolicyKey(account.String()))
}

func (k Keeper) HasPolicy(ctx sdk.Context, account sdk.AccAddress) bool {
	_, found := k.GetPolicy(ctx, account)
	return found
}
