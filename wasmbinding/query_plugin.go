package wasmbinding

import (
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/Daviddochain/dochain-core/v4/wasmbinding/bindings"
	marketkeeper "github.com/Daviddochain/dochain-core/v4/x/market/keeper"
	markettypes "github.com/Daviddochain/dochain-core/v4/x/market/types"
	oracletypes "github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	treasurytypes "github.com/Daviddochain/dochain-core/v4/x/treasury/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// TaxCapQueryResponse - tax cap query response for wasm module
type TaxCapQueryResponse struct {
	// uint64 string, eg "1000000"
	Cap string `json:"cap"`
}

// StargateQuerier dispatches whitelisted stargate queries
func StargateQuerier(
	queryRouter baseapp.GRPCQueryRouter,
	cdc codec.Codec,
) func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	return stargateQuerierWithQueryServers(queryRouter, cdc, nil, nil)
}

func stargateQuerierWithQueryServers(
	queryRouter baseapp.GRPCQueryRouter,
	cdc codec.Codec,
	marketQueryServer markettypes.QueryServer,
	treasuryQueryServer treasurytypes.QueryServer,
) func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		queryPath := canonicalStargateQueryPath(request.Path)
		protoResponseType, err := GetWhitelistedQuery(queryPath)
		if err != nil {
			return nil, err
		}

		route := queryRouter.Route(queryPath)
		if route == nil {
			response, handled, err := queryNativeStargate(
				ctx,
				queryPath,
				request.Data,
				cdc,
				marketQueryServer,
				treasuryQueryServer,
			)
			if handled {
				return response, err
			}
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", queryPath)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: queryPath,
		})
		if err != nil {
			return nil, err
		}

		bz, err := ConvertProtoToJSONMarshal(protoResponseType, res.Value, cdc)
		if err != nil {
			return nil, err
		}

		return bz, nil
	}
}

func queryNativeStargate(
	ctx sdk.Context,
	queryPath string,
	data []byte,
	cdc codec.Codec,
	marketQueryServer markettypes.QueryServer,
	treasuryQueryServer treasurytypes.QueryServer,
) ([]byte, bool, error) {
	switch queryPath {
	case "/do.market.v1beta1.Query/Swap":
		if marketQueryServer == nil {
			return nil, false, nil
		}
		var req markettypes.QuerySwapRequest
		if err := cdc.Unmarshal(data, &req); err != nil {
			return nil, true, wasmvmtypes.Unknown{}
		}
		response, err := marketQueryServer.Swap(sdk.WrapSDKContext(ctx), &req)
		return marshalNativeStargateResponse(cdc, response, err)
	case "/do.treasury.v1beta1.Query/TaxRate":
		if treasuryQueryServer == nil {
			return nil, false, nil
		}
		var req treasurytypes.QueryTaxRateRequest
		if err := cdc.Unmarshal(data, &req); err != nil {
			return nil, true, wasmvmtypes.Unknown{}
		}
		response, err := treasuryQueryServer.TaxRate(sdk.WrapSDKContext(ctx), &req)
		return marshalNativeStargateResponse(cdc, response, err)
	case "/do.treasury.v1beta1.Query/TaxCap":
		if treasuryQueryServer == nil {
			return nil, false, nil
		}
		var req treasurytypes.QueryTaxCapRequest
		if err := cdc.Unmarshal(data, &req); err != nil {
			return nil, true, wasmvmtypes.Unknown{}
		}
		response, err := treasuryQueryServer.TaxCap(sdk.WrapSDKContext(ctx), &req)
		return marshalNativeStargateResponse(cdc, response, err)
	default:
		return nil, false, nil
	}
}

func marshalNativeStargateResponse(
	cdc codec.Codec,
	response codec.ProtoMarshaler,
	queryErr error,
) ([]byte, bool, error) {
	if queryErr != nil {
		return nil, true, queryErr
	}
	bz, err := cdc.MarshalJSON(response)
	if err != nil {
		return nil, true, wasmvmtypes.Unknown{}
	}
	return bz, true, nil
}

// normalizeLegacyRoutedQueryJSON transforms legacy routed shape
// {"route":"treasury|oracle","query_data":{...}}
// into the modern flat DoQuery JSON understood by bindings.DoQuery.
// If the request is not a legacy routed query or cannot be normalized,
// the original request is returned unchanged.
func normalizeLegacyRoutedQueryJSON(request json.RawMessage) json.RawMessage {
	type legacyRouted struct {
		Route     string                     `json:"route"`
		QueryData map[string]json.RawMessage `json:"query_data"`
	}

	// limit request size to 64kb to check for legacy (DoS)
	if len(request) > 64<<10 {
		return request
	}

	var lr legacyRouted
	// if it cannot be unmarshaled into legacyRouted, treat as modern DoQuery
	if err := json.Unmarshal(request, &lr); err != nil || lr.Route == "" {
		return request
	}

	switch lr.Route {
	case treasurytypes.ModuleName:
		if _, ok := lr.QueryData["tax_rate"]; ok {
			// modern tax_rate has empty object
			if bz, err := json.Marshal(map[string]any{"tax_rate": struct{}{}}); err == nil {
				return bz
			}
		}
		if capRaw, ok := lr.QueryData["tax_cap"]; ok {
			// pass inner as-is (object with denom expected by old callers)
			if bz, err := json.Marshal(map[string]json.RawMessage{"tax_cap": capRaw}); err == nil {
				return bz
			}
		}
	case oracletypes.ModuleName:
		if er, ok := lr.QueryData["exchange_rates"]; ok {
			// pass inner as-is (expects {base_denom, quote_denoms})
			if bz, err := json.Marshal(map[string]json.RawMessage{"exchange_rates": er}); err == nil {
				return bz
			}
		}
	case markettypes.ModuleName:
		if sw, ok := lr.QueryData["swap"]; ok {
			// pass inner as-is ({offer_coin, ask_denom})
			if bz, err := json.Marshal(map[string]json.RawMessage{"swap": sw}); err == nil {
				return bz
			}
		}
	}

	// none of the legacy routes matched, return original request
	return request
}

// CustomQuerier dispatches custom CosmWasm bindings queries.
func CustomQuerier(qp *QueryPlugin) func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		normalized := normalizeLegacyRoutedQueryJSON(request)
		var contractQuery bindings.DoQuery
		if err := json.Unmarshal(normalized, &contractQuery); err != nil {
			return nil, errorsmod.Wrap(err, "do query")
		}

		switch {
		case contractQuery.Swap != nil:
			q := marketkeeper.NewQuerier(*qp.marketKeeper)
			res, err := q.Swap(sdk.WrapSDKContext(ctx), &markettypes.QuerySwapRequest{
				OfferCoin: contractQuery.Swap.OfferCoin.String(),
				AskDenom:  contractQuery.Swap.AskDenom,
			})
			if err != nil {
				return nil, err
			}

			bz, err := json.Marshal(bindings.SwapQueryResponse{Receive: ConvertSdkCoinToWasmCoin(res.ReturnCoin)})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		case contractQuery.ExchangeRates != nil:
			// dotest / BASE_DENOM
			baseDenomExchangeRate, err := qp.oracleKeeper.GetDoExchangeRate(ctx, contractQuery.ExchangeRates.BaseDenom)
			if err != nil {
				return nil, err
			}

			var items []bindings.ExchangeRateItem
			for _, quoteDenom := range contractQuery.ExchangeRates.QuoteDenoms {
				// dotest / QUOTE_DENOM
				quoteDenomExchangeRate, err := qp.oracleKeeper.GetDoExchangeRate(ctx, quoteDenom)
				if err != nil {
					continue
				}

				// (dotest / QUOTE_DENOM) / (BASE_DENOM / dotest) = BASE_DENOM / QUOTE_DENOM
				items = append(items, bindings.ExchangeRateItem{
					ExchangeRate: quoteDenomExchangeRate.Quo(baseDenomExchangeRate).String(),
					QuoteDenom:   quoteDenom,
				})
			}

			bz, err := json.Marshal(bindings.ExchangeRatesQueryResponse{
				BaseDenom:     contractQuery.ExchangeRates.BaseDenom,
				ExchangeRates: items,
			})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		case contractQuery.TaxRate != nil:
			taxRate := qp.treasuryKeeper.GetTaxRate(ctx)
			bz, err := json.Marshal(bindings.TaxRateQueryResponse{Rate: taxRate.String()})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		case contractQuery.TaxCap != nil:
			taxCap := qp.treasuryKeeper.GetTaxCap(ctx, contractQuery.TaxCap.Denom)
			bz, err := json.Marshal(TaxCapQueryResponse{Cap: taxCap.String()})
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
			}

			return bz, nil

		default:
			return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown do query variant"}
		}
	}
}
