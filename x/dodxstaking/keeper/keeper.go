package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper stores DODx governance stakes.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	AccountKeeper types.AccountKeeper
	BankKeeper    types.BankKeeper
}

// NewKeeper creates a DODx staking keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetStake returns the staked DODx coin for an address.
func (k Keeper) GetStake(ctx sdk.Context, addr sdk.AccAddress) sdk.Coin {
	return sdk.NewCoin(core.MicroDODxDenom, k.GetStakeAmount(ctx, addr))
}

// GetStakeAmount returns the staked DODx amount for an address.
func (k Keeper) GetStakeAmount(ctx sdk.Context, addr sdk.AccAddress) sdkmath.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(stakeKey(addr))
	if bz == nil {
		return sdkmath.ZeroInt()
	}

	ip := sdk.IntProto{}
	k.cdc.MustUnmarshal(bz, &ip)
	return ip.Int
}

// SetStakeAmount stores the staked DODx amount for an address.
func (k Keeper) SetStakeAmount(ctx sdk.Context, addr sdk.AccAddress, amount sdkmath.Int) {
	store := ctx.KVStore(k.storeKey)
	key := stakeKey(addr)
	if amount.IsZero() {
		store.Delete(key)
		return
	}

	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: amount})
	store.Set(key, bz)
}

// GetTotalStaked returns total staked DODx.
func (k Keeper) GetTotalStaked(ctx sdk.Context) sdk.Coin {
	return sdk.NewCoin(core.MicroDODxDenom, k.GetTotalStakedAmount(ctx))
}

// GetTotalStakedAmount returns total staked DODx amount.
func (k Keeper) GetTotalStakedAmount(ctx sdk.Context) sdkmath.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalStakedKey)
	if bz == nil {
		return sdkmath.ZeroInt()
	}

	ip := sdk.IntProto{}
	k.cdc.MustUnmarshal(bz, &ip)
	return ip.Int
}

// SetTotalStakedAmount stores total staked DODx.
func (k Keeper) SetTotalStakedAmount(ctx sdk.Context, amount sdkmath.Int) {
	store := ctx.KVStore(k.storeKey)
	if amount.IsZero() {
		store.Delete(types.TotalStakedKey)
		return
	}

	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: amount})
	store.Set(types.TotalStakedKey, bz)
}

// AddStake increases one account stake and total stake.
func (k Keeper) AddStake(ctx sdk.Context, addr sdk.AccAddress, amount sdkmath.Int) {
	k.SetStakeAmount(ctx, addr, k.GetStakeAmount(ctx, addr).Add(amount))
	k.SetTotalStakedAmount(ctx, k.GetTotalStakedAmount(ctx).Add(amount))
}

// RemoveStake decreases one account stake and total stake.
func (k Keeper) RemoveStake(ctx sdk.Context, addr sdk.AccAddress, amount sdkmath.Int) error {
	current := k.GetStakeAmount(ctx, addr)
	if current.LT(amount) {
		return types.ErrInsufficientStake
	}

	k.SetStakeAmount(ctx, addr, current.Sub(amount))
	k.SetTotalStakedAmount(ctx, k.GetTotalStakedAmount(ctx).Sub(amount))
	return nil
}

// IterateStakes iterates all non-zero account stakes.
func (k Keeper) IterateStakes(ctx sdk.Context, handler func(addr sdk.AccAddress, amount sdkmath.Int) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.StakeKeyPrefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr := sdk.AccAddress(iterator.Key()[1:])
		ip := sdk.IntProto{}
		k.cdc.MustUnmarshal(iterator.Value(), &ip)
		if handler(addr, ip.Int) {
			break
		}
	}
}

// SetGovernanceEnabled records whether DODx governance tallying is active.
func (k Keeper) SetGovernanceEnabled(ctx sdk.Context, enabled bool) {
	store := ctx.KVStore(k.storeKey)
	if !enabled {
		store.Delete(types.GovernanceEnabledKey)
		return
	}
	store.Set(types.GovernanceEnabledKey, types.GovernanceEnabledFlag)
}

// GovernanceEnabled reports whether DODx governance tallying is active.
func (k Keeper) GovernanceEnabled(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GovernanceEnabledKey)
}

func stakeKey(addr sdk.AccAddress) []byte {
	key := make([]byte, 1+len(addr))
	key[0] = types.StakeKeyPrefix
	copy(key[1:], addr)
	return key
}
