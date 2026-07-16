package keeper

import (
	"bytes"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/market/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Keeper of the market store.
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramstypes.Subspace

	AccountKeeper types.AccountKeeper
	BankKeeper    types.BankKeeper
	OracleKeeper  types.OracleKeeper
}

// NewKeeper constructs a new keeper for the market module.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	paramstore paramstypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
) Keeper {
	// Ensure the market module account is set.
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	// Set KeyTable if it has not already been set.
	if !paramstore.HasKeyTable() {
		paramstore = paramstore.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramSpace:    paramstore,
		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
		OracleKeeper:  oracleKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetDoPoolDelta returns the gap between the Do pool and the base pool.
func (k Keeper) GetDoPoolDelta(ctx sdk.Context) math.LegacyDec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DoPoolDeltaKey)
	if bz == nil {
		return math.LegacyZeroDec()
	}

	dp := sdk.DecProto{}
	k.cdc.MustUnmarshal(bz, &dp)
	return dp.Dec
}

// SetDoPoolDelta updates the Do pool delta, which is the gap between the Do pool
// and the configured base pool.
func (k Keeper) SetDoPoolDelta(ctx sdk.Context, delta math.LegacyDec) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&sdk.DecProto{Dec: delta})
	if ctx.BlockHeight() >= core.ReplayWriteOptimizationHeight && bytes.Equal(store.Get(types.DoPoolDeltaKey), bz) {
		return
	}
	store.Set(types.DoPoolDeltaKey, bz)
}

// ReplenishPools moves the pool state back toward the configured base pool over time.
func (k Keeper) ReplenishPools(ctx sdk.Context) {
	poolDelta := k.GetDoPoolDelta(ctx)

	poolRecoveryPeriod := int64(k.PoolRecoveryPeriod(ctx))
	poolRegressionAmt := poolDelta.QuoInt64(poolRecoveryPeriod)

	// Replenish pools toward the configured base pool.
	// regressionAmt cannot make delta exactly zero in a single step unless the math does so naturally.
	poolDelta = poolDelta.Sub(poolRegressionAmt)

	k.SetDoPoolDelta(ctx, poolDelta)
}
