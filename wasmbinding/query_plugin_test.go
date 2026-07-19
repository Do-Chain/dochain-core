package wasmbinding

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	markettypes "github.com/Daviddochain/dochain-core/v4/x/market/types"
	treasurytypes "github.com/Daviddochain/dochain-core/v4/x/treasury/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

type nativeMarketQueryServer struct {
	markettypes.UnimplementedQueryServer
}

func (nativeMarketQueryServer) Swap(
	context.Context,
	*markettypes.QuerySwapRequest,
) (*markettypes.QuerySwapResponse, error) {
	return &markettypes.QuerySwapResponse{ReturnCoin: sdk.NewInt64Coin("uusd", 7)}, nil
}

type nativeTreasuryQueryServer struct {
	treasurytypes.UnimplementedQueryServer
}

func (nativeTreasuryQueryServer) TaxRate(
	context.Context,
	*treasurytypes.QueryTaxRateRequest,
) (*treasurytypes.QueryTaxRateResponse, error) {
	return &treasurytypes.QueryTaxRateResponse{
		TaxRate: sdkmath.LegacyMustNewDecFromStr("0.125"),
	}, nil
}

func (nativeTreasuryQueryServer) TaxCap(
	context.Context,
	*treasurytypes.QueryTaxCapRequest,
) (*treasurytypes.QueryTaxCapResponse, error) {
	return &treasurytypes.QueryTaxCapResponse{TaxCap: sdkmath.NewInt(99)}, nil
}

func TestQueryNativeStargateHandlesAllowlistedQueriesWithoutGRPCRoutes(t *testing.T) {
	appCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	marketQueryServer := &nativeMarketQueryServer{}
	treasuryQueryServer := &nativeTreasuryQueryServer{}

	t.Run("market swap", func(t *testing.T) {
		requestBz := appCodec.MustMarshal(&markettypes.QuerySwapRequest{
			OfferCoin: "1udo",
			AskDenom:  "uusd",
		})
		responseBz, handled, err := queryNativeStargate(
			sdk.Context{},
			"/do.market.v1beta1.Query/Swap",
			requestBz,
			appCodec,
			marketQueryServer,
			treasuryQueryServer,
		)
		require.NoError(t, err)
		require.True(t, handled)

		var response markettypes.QuerySwapResponse
		appCodec.MustUnmarshalJSON(responseBz, &response)
		require.Equal(t, sdk.NewInt64Coin("uusd", 7), response.ReturnCoin)
	})

	t.Run("treasury tax rate", func(t *testing.T) {
		requestBz := appCodec.MustMarshal(&treasurytypes.QueryTaxRateRequest{})
		responseBz, handled, err := queryNativeStargate(
			sdk.Context{},
			"/do.treasury.v1beta1.Query/TaxRate",
			requestBz,
			appCodec,
			marketQueryServer,
			treasuryQueryServer,
		)
		require.NoError(t, err)
		require.True(t, handled)

		var response treasurytypes.QueryTaxRateResponse
		appCodec.MustUnmarshalJSON(responseBz, &response)
		require.Equal(t, sdkmath.LegacyMustNewDecFromStr("0.125"), response.TaxRate)
	})

	t.Run("treasury tax cap", func(t *testing.T) {
		requestBz := appCodec.MustMarshal(&treasurytypes.QueryTaxCapRequest{Denom: "uusd"})
		responseBz, handled, err := queryNativeStargate(
			sdk.Context{},
			"/do.treasury.v1beta1.Query/TaxCap",
			requestBz,
			appCodec,
			marketQueryServer,
			treasuryQueryServer,
		)
		require.NoError(t, err)
		require.True(t, handled)

		var response treasurytypes.QueryTaxCapResponse
		appCodec.MustUnmarshalJSON(responseBz, &response)
		require.Equal(t, sdkmath.NewInt(99), response.TaxCap)
	})

	_, handled, err := queryNativeStargate(
		sdk.Context{},
		"/do.unknown.v1.Query/Anything",
		nil,
		appCodec,
		marketQueryServer,
		treasuryQueryServer,
	)
	require.NoError(t, err)
	require.False(t, handled)
}
