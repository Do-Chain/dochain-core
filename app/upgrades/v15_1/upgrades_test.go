package v15_1

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/stretchr/testify/require"
)

func TestV151UsesCanonicalMainnetChainID(t *testing.T) {
	require.Equal(t, "Do-Chain", core.DoChainMainnetChainID)
	require.Equal(t, core.DoChainMainnetChainID, doChainID)
}

func TestWasmPermissionsDefaultClosedUnlessAllowlisted(t *testing.T) {
	originalAllowlist := cosmWasmUploadAllowlist
	t.Cleanup(func() { cosmWasmUploadAllowlist = originalAllowlist })

	cosmWasmUploadAllowlist = nil
	access := wasmUploadAccessConfig()
	require.Equal(t, wasmtypes.AccessTypeNobody, access.Permission)
	require.Empty(t, access.Addresses)

	cosmWasmUploadAllowlist = []string{"do1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"}
	access = wasmUploadAccessConfig()
	require.Equal(t, wasmtypes.AccessTypeAnyOfAddresses, access.Permission)
	require.Equal(t, cosmWasmUploadAllowlist, access.Addresses)
}

func TestDelegatorWideSlashingDisabled(t *testing.T) {
	params := slashingtypes.DefaultParams()
	require.False(t, params.SlashFractionDowntime.IsZero())
	require.False(t, params.SlashFractionDoubleSign.IsZero())

	secured := delegatorWideSlashingDisabled(params)
	require.Equal(t, sdkmath.LegacyZeroDec(), secured.SlashFractionDowntime)
	require.Equal(t, sdkmath.LegacyZeroDec(), secured.SlashFractionDoubleSign)
	require.NoError(t, secured.Validate())
}
