package wasmbinding

import (
	"fmt"
	"reflect"
	"sync"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	markettypes "github.com/Daviddochain/dochain-core/v4/x/market/types"
	oracletypes "github.com/Daviddochain/dochain-core/v4/x/oracle/types"
	treasurytypes "github.com/Daviddochain/dochain-core/v4/x/treasury/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// stargateWhitelist keeps whitelist and its deterministic
// response binding for stargate queries.
//
// The query can be multi-thread, so we have to use
// thread safe sync.Map.
var stargateWhitelist sync.Map

// legacyStargateQueryAliases retains wire-compatible Terra query paths used by
// contracts deployed before the Do protobuf namespace was introduced. The
// aliases resolve only to queries that are already in the Do allowlist.
var legacyStargateQueryAliases = map[string]string{
	"/terra.market.v1beta1.Query/Swap":         "/do.market.v1beta1.Query/Swap",
	"/terra.oracle.v1beta1.Query/ExchangeRate": "/do.oracle.v1beta1.Query/ExchangeRate",
	"/terra.treasury.v1beta1.Query/TaxCap":     "/do.treasury.v1beta1.Query/TaxCap",
	"/terra.treasury.v1beta1.Query/TaxRate":    "/do.treasury.v1beta1.Query/TaxRate",
}

func init() {
	// market
	setWhitelistedQuery("/do.market.v1beta1.Query/Swap", &markettypes.QuerySwapResponse{})

	// treasury
	setWhitelistedQuery("/do.treasury.v1beta1.Query/TaxCap", &treasurytypes.QueryTaxCapResponse{})
	setWhitelistedQuery("/do.treasury.v1beta1.Query/TaxRate", &treasurytypes.QueryTaxRateResponse{})

	// oracle
	setWhitelistedQuery("/do.oracle.v1beta1.Query/ExchangeRate", &oracletypes.QueryExchangeRateResponse{})
}

// GetWhitelistedQuery returns the whitelisted query at the provided path.
// If the query does not exist, or it was setup wrong by the chain, this returns an error.
func GetWhitelistedQuery(queryPath string) (codec.ProtoMarshaler, error) {
	queryPath = canonicalStargateQueryPath(queryPath)
	protoResponseAny, isWhitelisted := stargateWhitelist.Load(queryPath)
	if !isWhitelisted {
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", queryPath)}
	}
	protoResponsePrototype, ok := protoResponseAny.(codec.ProtoMarshaler)
	if !ok {
		return nil, wasmvmtypes.Unknown{}
	}
	prototypeType := reflect.TypeOf(protoResponsePrototype)
	if prototypeType.Kind() != reflect.Ptr {
		return nil, wasmvmtypes.Unknown{}
	}
	protoResponseType, ok := reflect.New(prototypeType.Elem()).Interface().(codec.ProtoMarshaler)
	if !ok {
		return nil, wasmvmtypes.Unknown{}
	}
	return protoResponseType, nil
}

func canonicalStargateQueryPath(queryPath string) string {
	if canonicalPath, ok := legacyStargateQueryAliases[queryPath]; ok {
		return canonicalPath
	}
	return queryPath
}

func setWhitelistedQuery(queryPath string, protoType codec.ProtoMarshaler) {
	stargateWhitelist.Store(queryPath, protoType)
}
