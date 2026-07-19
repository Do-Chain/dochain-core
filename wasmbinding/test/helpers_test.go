package wasmbinding_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	apptesting "github.com/Daviddochain/dochain-core/v4/app/testing"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestWasmFixtureIntegrity(t *testing.T) {
	fixtures := map[string]string{
		DoBindingsPath:          "2fb2553dee6a3790da60ba23e8602b487a88699581eed6d7d6eee06bec191448",
		DoRenovatedBindingsPath: "5b1f9519633884b84f40f1e2404a145950bfeefbedfa42074c8b9dfc850f8a02",
		DoStargateQueryPath:     "232d2b51bc66ee76b3cae8a51ed8462f970c6da593401625a6fc222dcd12b6bd",
	}
	for path, expectedHash := range fixtures {
		t.Run(path, func(t *testing.T) {
			wasmCode, err := os.ReadFile(path)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(wasmCode), 4)
			require.Equal(t, []byte{0x00, 0x61, 0x73, 0x6d}, wasmCode[:4])
			require.Equal(t, expectedHash, fmt.Sprintf("%x", sha256.Sum256(wasmCode)))
		})
	}
}

type WasmTestSuite struct {
	apptesting.KeeperTestHelper
}

func TestWasmTestSuite(t *testing.T) {
	if !apptesting.WasmVMAvailable {
		t.Skip("Wasm binding tests require a CGO-enabled WasmVM build")
	}
	suite.Run(t, new(WasmTestSuite))
}

func (s *WasmTestSuite) SetupTest() {
	s.Setup(s.T(), apptesting.SimAppChainID)
}

func (s *WasmTestSuite) InstantiateContract(addr sdk.AccAddress, contractPath string) sdk.AccAddress {
	wasmKeeper := s.App.WasmKeeper

	codeID := s.storeReflectCode(addr, contractPath)

	cInfo := wasmKeeper.GetCodeInfo(s.Ctx, codeID)
	s.Require().NotNil(cInfo)

	contractAddr := s.instantiateContract(addr, codeID)

	// check if contract is instantiated
	info := wasmKeeper.GetContractInfo(s.Ctx, contractAddr)
	s.Require().NotNil(info)

	return contractAddr
}

func (s *WasmTestSuite) storeReflectCode(addr sdk.AccAddress, contractPath string) uint64 {
	wasmCode, err := os.ReadFile(contractPath)
	s.Require().NoError(err)

	codeID, _, err := wasmkeeper.NewDefaultPermissionKeeper(s.App.WasmKeeper).Create(s.Ctx, addr, wasmCode, &wasmtypes.AllowEverybody)
	s.Require().NoError(err)

	return codeID
}

func (s *WasmTestSuite) instantiateContract(funder sdk.AccAddress, codeID uint64) sdk.AccAddress {
	initMsgBz := []byte("{}")
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(s.App.WasmKeeper)
	addr, _, err := contractKeeper.Instantiate(s.Ctx, codeID, funder, funder, initMsgBz, "label", nil)
	s.Require().NoError(err)

	return addr
}
