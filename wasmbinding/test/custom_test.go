package wasmbinding_test

import (
	"github.com/Daviddochain/dochain-core/v4/wasmbinding/bindings"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DoBindingsPath          = "../testdata/terra_reflect.wasm"
	DoRenovatedBindingsPath = "../testdata/old/bindings_tester.wasm"
	DoStargateQueryPath     = "../testdata/stargate_tester.wasm"
)

// go test -v -run ^TestWasmTestSuite/TestExecuteBindingsAll$ github.com/Daviddochain/dochain-core/v4/wasmbinding/test
func (s *WasmTestSuite) TestExecuteBindingsAll() {
	cases := []struct {
		name        string
		path        string
		executeFunc func(contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.DoMsg, funds sdk.Coin) error
		queryFunc   func(contract sdk.AccAddress, request bindings.DoQuery, response interface{})
	}{
		{
			name:        "do",
			path:        DoBindingsPath,
			executeFunc: s.executeCustom,
			queryFunc:   s.queryCustom,
		},
		{
			name:        "Old do bindings",
			path:        DoRenovatedBindingsPath,
			executeFunc: s.executeOldBindings,
			queryFunc:   s.queryOldBindings,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			// Msg
			s.Run("TestSwap", func() {
				s.Swap(tc.path, tc.executeFunc)
			})
			s.Run("TestSwapSend", func() {
				s.SwapSend(tc.path, tc.executeFunc)
			})
		})
	}
}

// go test -v -run ^TestWasmTestSuite/TestQueryBindingsAll$ github.com/Daviddochain/dochain-core/v4/wasmbinding/test
func (s *WasmTestSuite) TestQueryBindingsAll() {
	cases := []struct {
		name        string
		path        string
		executeFunc func(contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.DoMsg, funds sdk.Coin) error
		queryFunc   func(contract sdk.AccAddress, request bindings.DoQuery, response interface{})
	}{
		{
			name:        "do",
			path:        DoBindingsPath,
			executeFunc: s.executeCustom,
			queryFunc:   s.queryCustom,
		},
		{
			name:        "Old do bindings",
			path:        DoRenovatedBindingsPath,
			executeFunc: s.executeOldBindings,
			queryFunc:   s.queryOldBindings,
		},
		{
			name:        "do Stargate",
			path:        DoStargateQueryPath,
			executeFunc: nil,
			queryFunc:   s.queryStargate,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			// Query
			s.Run("TestQuerySwap", func() {
				s.QuerySwap(tc.path, tc.queryFunc)
			})
			s.Run("TestQueryExchangeRates", func() {
				s.QueryExchangeRates(tc.path, tc.queryFunc)
			})
			s.Run("TestQueryTaxRate", func() {
				s.QueryTaxRate(tc.path, tc.queryFunc)
			})
			s.Run("TestQueryTaxCap", func() {
				s.QueryTaxCap(tc.path, tc.queryFunc)
			})
		})
	}
}
