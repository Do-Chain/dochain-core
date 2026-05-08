package market

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/market/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/market/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMarketFilters(t *testing.T) {
	input, h := setup(t)

	// Case 2: Normal MsgSwap submission goes through
	offerCoin := sdk.NewCoin(core.MicroDoDenom, sdkmath.NewInt(10))
	prevoteMsg := types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroSDRDenom)
	_, err := h.Swap(sdk.WrapSDKContext(input.Ctx), prevoteMsg)
	require.NoError(t, err)
}

func TestSwapMsg_FailZeroReturn(t *testing.T) {
	input, h := setup(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	params.MinStabilitySpread = sdkmath.LegacyOneDec()
	input.MarketKeeper.SetParams(input.Ctx, params)

	amt := sdkmath.NewInt(10)
	offerCoin := sdk.NewCoin(core.MicroDoDenom, amt)
	swapMsg := types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroSDRDenom)
	_, err := h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.Error(t, err)
}

func TestSwapMsg(t *testing.T) {
	input, h := setup(t)

	params := input.MarketKeeper.GetParams(input.Ctx)
	params.MinStabilitySpread = sdkmath.LegacyZeroDec()
	input.MarketKeeper.SetParams(input.Ctx, params)

	beforeDoPoolDelta := input.MarketKeeper.GetDoPoolDelta(input.Ctx)

	amt := sdkmath.NewInt(10)
	offerCoin := sdk.NewCoin(core.MicroDoDenom, amt)
	swapMsg := types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroSDRDenom)
	_, err := h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err)

	afterDoPoolDelta := input.MarketKeeper.GetDoPoolDelta(input.Ctx)
	diff := beforeDoPoolDelta.Sub(afterDoPoolDelta)

	// calculate estimation
	basePool := input.MarketKeeper.GetParams(input.Ctx).BasePool
	cp := basePool.Mul(basePool)

	doPool := basePool.Add(beforeDoPoolDelta)
	baseAssetPool := cp.Quo(doPool)
	baseOfferAmount := sdkmath.LegacyNewDecFromInt(amt)
	estmiatedDiff := doPool.Sub(cp.Quo(baseAssetPool.Add(baseOfferAmount)))
	require.True(t, estmiatedDiff.Sub(diff.Abs()).LTE(sdkmath.LegacyNewDecWithPrec(1, 6)))

	// invalid recursive swap
	swapMsg = types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroDoDenom)

	_, err = h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.Error(t, err)

	// valid zero tobin tax test
	input.OracleKeeper.SetTobinTax(input.Ctx, core.MicroKRWDenom, sdkmath.LegacyZeroDec())
	input.OracleKeeper.SetTobinTax(input.Ctx, core.MicroSDRDenom, sdkmath.LegacyZeroDec())
	offerCoin = sdk.NewCoin(core.MicroSDRDenom, amt)
	swapMsg = types.NewMsgSwap(keeper.Addrs[0], offerCoin, core.MicroKRWDenom)
	_, err = h.Swap(sdk.WrapSDKContext(input.Ctx), swapMsg)
	require.NoError(t, err)
}

func TestSwapSendMsg(t *testing.T) {
	input, h := setup(t)

	amt := sdkmath.NewInt(10)
	offerCoin := sdk.NewCoin(core.MicroDoDenom, amt)
	retCoin, spread, err := input.MarketKeeper.ComputeSwap(input.Ctx, offerCoin, core.MicroSDRDenom)
	require.NoError(t, err)

	expectedAmt := retCoin.Amount.Mul(sdkmath.LegacyOneDec().Sub(spread)).TruncateInt()

	swapSendMsg := types.NewMsgSwapSend(keeper.Addrs[0], keeper.Addrs[1], offerCoin, core.MicroSDRDenom)
	_, err = h.SwapSend(sdk.WrapSDKContext(input.Ctx), swapSendMsg)
	require.NoError(t, err)

	balance := input.BankKeeper.GetBalance(input.Ctx, keeper.Addrs[1], core.MicroSDRDenom)
	require.Equal(t, expectedAmt, balance.Amount)
}
