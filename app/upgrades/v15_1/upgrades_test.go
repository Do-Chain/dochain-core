package v15_1

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/stretchr/testify/require"
)

func TestV151UsesCanonicalMainnetChainID(t *testing.T) {
	require.Equal(t, "Do-Chain", core.DoChainMainnetChainID)
	require.Equal(t, core.DoChainMainnetChainID, doChainID)
}

func TestHistoricalWasmPermissionsRemainReplayCompatible(t *testing.T) {
	originalAllowlist := cosmWasmUploadAllowlist
	t.Cleanup(func() { cosmWasmUploadAllowlist = originalAllowlist })

	cosmWasmUploadAllowlist = nil
	access := wasmUploadAccessConfig()
	require.Equal(t, wasmtypes.AccessTypeEverybody, access.Permission)
	require.Empty(t, access.Addresses)

	cosmWasmUploadAllowlist = []string{"do1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"}
	access = wasmUploadAccessConfig()
	require.Equal(t, wasmtypes.AccessTypeAnyOfAddresses, access.Permission)
	require.Equal(t, cosmWasmUploadAllowlist, access.Addresses)
}
