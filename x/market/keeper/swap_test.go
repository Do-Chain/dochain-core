package keeper

import (
	"math/rand"
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestApplySwapToPool(t *testing.T) {
	input := CreateTestInput(t)

	doPriceInSDR := sdkmath.LegacyNewDecWithPrec(17, 1)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroSDRDenom, doPriceInSDR)

	offerCoin := sdk.NewCoin(core.MicroDoDenom, sdkmath.NewInt(1000))
	askCoin := sdk.NewDecCoin(core.MicroSDRDenom, sdkmath.NewInt(1700))
	oldSDRPoolDelta := input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	input.MarketKeeper.ApplySwapToPool(input.Ctx, offerCoin, askCoin)
	newSDRPoolDelta := input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	sdrDiff := newSDRPoolDelta.Sub(oldSDRPoolDelta)
	require.Equal(t, sdkmath.LegacyNewDec(-1000), sdrDiff)

	// reverse swap
	offerCoin = sdk.NewCoin(core.MicroSDRDenom, sdkmath.NewInt(1700))
	askCoin = sdk.NewDecCoin(core.MicroDoDenom, sdkmath.NewInt(1000))
	oldSDRPoolDelta = input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	input.MarketKeeper.ApplySwapToPool(input.Ctx, offerCoin, askCoin)
	newSDRPoolDelta = input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	sdrDiff = newSDRPoolDelta.Sub(oldSDRPoolDelta)
	require.Equal(t, sdkmath.LegacyNewDec(1000), sdrDiff)

	// do <> do, no pool changes are expected
	offerCoin = sdk.NewCoin(core.MicroSDRDenom, sdkmath.NewInt(1700))
	askCoin = sdk.NewDecCoin(core.MicroKRWDenom, sdkmath.NewInt(3400))
	oldSDRPoolDelta = input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	input.MarketKeeper.ApplySwapToPool(input.Ctx, offerCoin, askCoin)
	newSDRPoolDelta = input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	sdrDiff = newSDRPoolDelta.Sub(oldSDRPoolDelta)
	require.Equal(t, sdkmath.LegacyNewDec(0), sdrDiff)
}

func TestComputeSwap(t *testing.T) {
	input := CreateTestInput(t)

	// Set Oracle Price
	doPriceInSDR := sdkmath.LegacyNewDecWithPrec(17, 1)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroSDRDenom, doPriceInSDR)

	for i := 0; i < 100; i++ {
		swapAmountInSDR := doPriceInSDR.MulInt64(rand.Int63()%10000 + 2).TruncateInt()
		offerCoin := sdk.NewCoin(core.MicroSDRDenom, swapAmountInSDR)
		retCoin, spread, err := input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroDoDenom)

		require.NoError(t, err)
		require.True(t, spread.GTE(input.MarketKeeper.MinStabilitySpread(input.Ctx)))
		require.Equal(t, sdkmath.LegacyNewDecFromInt(offerCoin.Amount).Quo(doPriceInSDR), retCoin.Amount)
	}

	offerCoin := sdk.NewCoin(core.MicroSDRDenom, doPriceInSDR.QuoInt64(2).TruncateInt())
	_, _, err := input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroDoDenom)
	require.Error(t, err)
}

func TestComputeInternalSwap(t *testing.T) {
	input := CreateTestInput(t)

	// Set Oracle Price
	doPriceInSDR := sdkmath.LegacyNewDecWithPrec(17, 1)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroSDRDenom, doPriceInSDR)

	for i := 0; i < 100; i++ {
		offerCoin := sdk.NewDecCoin(core.MicroSDRDenom, doPriceInSDR.MulInt64(rand.Int63()+1).TruncateInt())
		retCoin, err := input.MarketKeeper.ComputeInternalSwap(input.Ctx, offerCoin, core.MicroDoDenom)
		require.NoError(t, err)
		require.Equal(t, offerCoin.Amount.Quo(doPriceInSDR), retCoin.Amount)
	}

	offerCoin := sdk.NewDecCoin(core.MicroSDRDenom, doPriceInSDR.QuoInt64(2).TruncateInt())
	_, err := input.MarketKeeper.ComputeInternalSwap(input.Ctx, offerCoin, core.MicroDoDenom)
	require.Error(t, err)
}

func TestIlliquidTobinTaxListParams(t *testing.T) {
	input := CreateTestInput(t)

	// Set Oracle Price
	doPriceInSDR := sdkmath.LegacyNewDecWithPrec(17, 1)
	doPriceInMNT := sdkmath.LegacyNewDecWithPrec(7652, 1)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroSDRDenom, doPriceInSDR)
	input.OracleKeeper.SetDoExchangeRate(input.Ctx, core.MicroMNTDenom, doPriceInMNT)

	tobinTax := sdkmath.LegacyNewDecWithPrec(25, 4)
	params := input.MarketKeeper.GetParams(input.Ctx)
	input.MarketKeeper.SetParams(input.Ctx, params)

	illiquidFactor := sdkmath.LegacyNewDec(2)
	input.OracleKeeper.SetTobinTax(input.Ctx, core.MicroSDRDenom, tobinTax)
	input.OracleKeeper.SetTobinTax(input.Ctx, core.MicroMNTDenom, tobinTax.Mul(illiquidFactor))

	swapAmountInSDR := doPriceInSDR.MulInt64(rand.Int63()%10000 + 2).TruncateInt()
	offerCoin := sdk.NewCoin(core.MicroSDRDenom, swapAmountInSDR)
	_, spread, err := input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroMNTDenom)
	require.NoError(t, err)
	require.Equal(t, tobinTax.Mul(illiquidFactor), spread)
}
