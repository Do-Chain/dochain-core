package keeper

import (
	"bytes"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RewardPrecision scales reward-per-staked-DODX accumulators. A staker's
// entitlement for a denom is stake * accumulator / RewardPrecision.
var RewardPrecision = sdkmath.NewInt(1_000_000_000_000_000_000)

func (k Keeper) getInt(ctx sdk.Context, key []byte) sdkmath.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(key)
	if bz == nil {
		return sdkmath.ZeroInt()
	}

	ip := sdk.IntProto{}
	k.cdc.MustUnmarshal(bz, &ip)
	return ip.Int
}

func (k Keeper) setInt(ctx sdk.Context, key []byte, amount sdkmath.Int) {
	store := ctx.KVStore(k.storeKey)
	if amount.IsZero() {
		store.Delete(key)
		return
	}

	bz := k.cdc.MustMarshal(&sdk.IntProto{Int: amount})
	store.Set(key, bz)
}

// GetRewardAccumulator returns the scaled cumulative rewards per staked DODX for a denom.
func (k Keeper) GetRewardAccumulator(ctx sdk.Context, denom string) sdkmath.Int {
	return k.getInt(ctx, rewardAccumulatorKey(denom))
}

// SetRewardAccumulator stores the scaled cumulative rewards per staked DODX for a denom.
func (k Keeper) SetRewardAccumulator(ctx sdk.Context, denom string, amount sdkmath.Int) {
	k.setInt(ctx, rewardAccumulatorKey(denom), amount)
	if amount.IsPositive() {
		k.setRewardDenom(ctx, denom)
	}
}

// GetRewardPoolAmount returns the accounted, unclaimed reward pool for a denom.
func (k Keeper) GetRewardPoolAmount(ctx sdk.Context, denom string) sdkmath.Int {
	return k.getInt(ctx, rewardPoolKey(denom))
}

// SetRewardPoolAmount stores the accounted, unclaimed reward pool for a denom.
func (k Keeper) SetRewardPoolAmount(ctx sdk.Context, denom string, amount sdkmath.Int) {
	k.setInt(ctx, rewardPoolKey(denom), amount)
	if amount.IsPositive() {
		k.setRewardDenom(ctx, denom)
	}
}

// GetRewardDebt returns the settled accumulator debt for an account and denom.
func (k Keeper) GetRewardDebt(ctx sdk.Context, addr sdk.AccAddress, denom string) sdkmath.Int {
	return k.getInt(ctx, rewardDebtKey(addr, denom))
}

// SetRewardDebt stores the settled accumulator debt for an account and denom.
func (k Keeper) SetRewardDebt(ctx sdk.Context, addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	k.setInt(ctx, rewardDebtKey(addr, denom), amount)
	if amount.IsPositive() {
		k.setRewardDenom(ctx, denom)
	}
}

// GetPendingRewardAmount returns the currently claimable stored reward for an account and denom.
func (k Keeper) GetPendingRewardAmount(ctx sdk.Context, addr sdk.AccAddress, denom string) sdkmath.Int {
	return k.getInt(ctx, pendingRewardKey(addr, denom))
}

// SetPendingRewardAmount stores the currently claimable reward for an account and denom.
func (k Keeper) SetPendingRewardAmount(ctx sdk.Context, addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	k.setInt(ctx, pendingRewardKey(addr, denom), amount)
	if amount.IsPositive() {
		k.setRewardDenom(ctx, denom)
	}
}

func (k Keeper) setRewardDenom(ctx sdk.Context, denom string) {
	ctx.KVStore(k.storeKey).Set(rewardDenomKey(denom), []byte{0x01})
}

// IterateRewardDenoms iterates all reward denoms that have been accounted.
func (k Keeper) IterateRewardDenoms(ctx sdk.Context, handler func(denom string) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.RewardDenomKeyPrefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key()[1:])
		if handler(denom) {
			break
		}
	}
}

// SyncRewardBalances accounts for native reward coins bank-sent directly to the
// dodxstaking module account, such as DEX pair fee splits. It excludes locked
// udodx staking principal and previously-accounted reward pools.
func (k Keeper) SyncRewardBalances(ctx sdk.Context) {
	moduleAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	if moduleAddr == nil {
		return
	}

	for _, coin := range k.BankKeeper.GetAllBalances(ctx, moduleAddr) {
		reserved := k.GetRewardPoolAmount(ctx, coin.Denom)
		if coin.Denom == core.MicroDODxDenom {
			reserved = reserved.Add(k.GetTotalStakedAmount(ctx))
		}
		if coin.Amount.GT(reserved) {
			delta := coin.Amount.Sub(reserved)
			k.CreditRewards(ctx, sdk.NewCoin(coin.Denom, delta))
		}
	}
}

// CreditRewards adds a reward coin to the pro-rata DODX-staker accumulator. If
// there is no DODX staked, the coin remains unaccounted in the module account and
// will be picked up by a later SyncRewardBalances call once staking exists.
func (k Keeper) CreditRewards(ctx sdk.Context, reward sdk.Coin) {
	if !reward.IsValid() || !reward.IsPositive() {
		return
	}

	totalStaked := k.GetTotalStakedAmount(ctx)
	if !totalStaked.IsPositive() {
		return
	}

	increment := reward.Amount.Mul(RewardPrecision).Quo(totalStaked)
	if !increment.IsPositive() {
		return
	}

	acc := k.GetRewardAccumulator(ctx, reward.Denom).Add(increment)
	k.SetRewardAccumulator(ctx, reward.Denom, acc)
	k.SetRewardPoolAmount(ctx, reward.Denom, k.GetRewardPoolAmount(ctx, reward.Denom).Add(reward.Amount))
	k.setRewardDenom(ctx, reward.Denom)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSyncRewards,
			sdk.NewAttribute(types.AttributeKeyDenom, reward.Denom),
			sdk.NewAttribute(types.AttributeKeyAmount, reward.String()),
		),
	)
}

// SettleRewards moves all rewards accrued by the current stake into pending
// balances and updates the account's reward debts.
func (k Keeper) SettleRewards(ctx sdk.Context, addr sdk.AccAddress) {
	stake := k.GetStakeAmount(ctx, addr)
	k.IterateRewardDenoms(ctx, func(denom string) bool {
		acc := k.GetRewardAccumulator(ctx, denom)
		entitled := stake.Mul(acc).Quo(RewardPrecision)
		debt := k.GetRewardDebt(ctx, addr, denom)
		if entitled.GT(debt) {
			pending := k.GetPendingRewardAmount(ctx, addr, denom).Add(entitled.Sub(debt))
			k.SetPendingRewardAmount(ctx, addr, denom, pending)
		}
		k.SetRewardDebt(ctx, addr, denom, entitled)
		return false
	})
}

// ResetRewardDebts resets reward debts after a stake amount change.
func (k Keeper) ResetRewardDebts(ctx sdk.Context, addr sdk.AccAddress) {
	stake := k.GetStakeAmount(ctx, addr)
	k.IterateRewardDenoms(ctx, func(denom string) bool {
		acc := k.GetRewardAccumulator(ctx, denom)
		k.SetRewardDebt(ctx, addr, denom, stake.Mul(acc).Quo(RewardPrecision))
		return false
	})
}

// PendingRewards returns claimable rewards, including unsettled accumulator
// rewards, without mutating state.
func (k Keeper) PendingRewards(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	stake := k.GetStakeAmount(ctx, addr)
	rewards := sdk.Coins{}
	k.IterateRewardDenoms(ctx, func(denom string) bool {
		pending := k.GetPendingRewardAmount(ctx, addr, denom)
		acc := k.GetRewardAccumulator(ctx, denom)
		entitled := stake.Mul(acc).Quo(RewardPrecision)
		debt := k.GetRewardDebt(ctx, addr, denom)
		if entitled.GT(debt) {
			pending = pending.Add(entitled.Sub(debt))
		}
		if pending.IsPositive() {
			rewards = rewards.Add(sdk.NewCoin(denom, pending))
		}
		return false
	})
	return rewards
}

// ClaimPendingRewards settles then clears pending rewards for the requested denoms.
// If denoms is empty, every available reward denom is claimed.
func (k Keeper) ClaimPendingRewards(ctx sdk.Context, addr sdk.AccAddress, denoms []string) (sdk.Coins, error) {
	k.SettleRewards(ctx, addr)

	if len(denoms) == 0 {
		k.IterateRewardDenoms(ctx, func(denom string) bool {
			denoms = append(denoms, denom)
			return false
		})
	}

	claimed := sdk.Coins{}
	seen := map[string]bool{}
	for _, denom := range denoms {
		if seen[denom] {
			continue
		}
		seen[denom] = true

		amount := k.GetPendingRewardAmount(ctx, addr, denom)
		if !amount.IsPositive() {
			continue
		}
		pool := k.GetRewardPoolAmount(ctx, denom)
		if pool.LT(amount) {
			return nil, types.ErrNoRewards
		}
		k.SetPendingRewardAmount(ctx, addr, denom, sdkmath.ZeroInt())
		k.SetRewardPoolAmount(ctx, denom, pool.Sub(amount))
		claimed = claimed.Add(sdk.NewCoin(denom, amount))
	}

	if claimed.Empty() {
		return nil, types.ErrNoRewards
	}

	return claimed, nil
}

// RewardPool returns all accounted, unclaimed reward pools.
func (k Keeper) RewardPool(ctx sdk.Context) sdk.Coins {
	pool := sdk.Coins{}
	k.IterateRewardDenoms(ctx, func(denom string) bool {
		amount := k.GetRewardPoolAmount(ctx, denom)
		if amount.IsPositive() {
			pool = pool.Add(sdk.NewCoin(denom, amount))
		}
		return false
	})
	return pool
}

// IterateRewardDebts iterates all per-account reward debt records.
func (k Keeper) IterateRewardDebts(ctx sdk.Context, handler func(addr sdk.AccAddress, denom string, amount sdkmath.Int) bool) {
	k.iterateAccountRewardAmounts(ctx, types.RewardDebtKeyPrefix, handler)
}

// IteratePendingRewardAmounts iterates all stored pending reward records.
func (k Keeper) IteratePendingRewardAmounts(ctx sdk.Context, handler func(addr sdk.AccAddress, denom string, amount sdkmath.Int) bool) {
	k.iterateAccountRewardAmounts(ctx, types.PendingRewardKeyPrefix, handler)
}

func (k Keeper) iterateAccountRewardAmounts(ctx sdk.Context, prefix byte, handler func(addr sdk.AccAddress, denom string, amount sdkmath.Int) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{prefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr, denom, ok := splitAccountDenomKey(iterator.Key())
		if !ok {
			continue
		}
		amount := k.getInt(ctx, iterator.Key())
		if handler(addr, denom, amount) {
			break
		}
	}
}

func rewardAccumulatorKey(denom string) []byte {
	return append([]byte{types.RewardAccumulatorKeyPrefix}, []byte(denom)...)
}

func rewardPoolKey(denom string) []byte {
	return append([]byte{types.RewardPoolKeyPrefix}, []byte(denom)...)
}

func rewardDenomKey(denom string) []byte {
	return append([]byte{types.RewardDenomKeyPrefix}, []byte(denom)...)
}

func rewardDebtKey(addr sdk.AccAddress, denom string) []byte {
	return accountDenomKey(types.RewardDebtKeyPrefix, addr, denom)
}

func pendingRewardKey(addr sdk.AccAddress, denom string) []byte {
	return accountDenomKey(types.PendingRewardKeyPrefix, addr, denom)
}

func accountDenomKey(prefix byte, addr sdk.AccAddress, denom string) []byte {
	var b bytes.Buffer
	b.WriteByte(prefix)
	b.WriteByte(byte(len(addr)))
	b.Write(addr)
	b.WriteString(denom)
	return b.Bytes()
}

func splitAccountDenomKey(key []byte) (sdk.AccAddress, string, bool) {
	if len(key) < 2 {
		return nil, "", false
	}
	addrLen := int(key[1])
	if len(key) < 2+addrLen {
		return nil, "", false
	}
	addr := sdk.AccAddress(key[2 : 2+addrLen])
	denom := string(key[2+addrLen:])
	if len(addr) == 0 || denom == "" {
		return nil, "", false
	}
	return addr, denom, true
}
