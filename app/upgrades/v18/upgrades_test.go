package v18

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"
)

func TestSecureWasmAccessParamsClosesBothPermissions(t *testing.T) {
	params := wasmtypes.DefaultParams()
	params.CodeUploadAccess = wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeEverybody}
	params.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody

	secured := secureWasmAccessParams(params)
	require.Equal(t, wasmtypes.AccessTypeNobody, secured.CodeUploadAccess.Permission)
	require.Empty(t, secured.CodeUploadAccess.Addresses)
	require.Equal(t, wasmtypes.AccessTypeNobody, secured.InstantiateDefaultPermission)
	require.NoError(t, secured.ValidateBasic())
}

func TestShouldSecureWasmInstantiateConfig(t *testing.T) {
	require.False(t, shouldSecureWasmInstantiateConfig(wasmtypes.CodeInfo{
		InstantiateConfig: wasmtypes.AllowNobody,
	}))
	require.True(t, shouldSecureWasmInstantiateConfig(wasmtypes.CodeInfo{
		InstantiateConfig: wasmtypes.AllowEverybody,
	}))
	require.True(t, shouldSecureWasmInstantiateConfig(wasmtypes.CodeInfo{
		InstantiateConfig: wasmtypes.AccessConfig{
			Permission: wasmtypes.AccessTypeAnyOfAddresses,
			Addresses:  []string{"do1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"},
		},
	}))
}

func TestV18HasNoStoreLayoutChanges(t *testing.T) {
	require.Empty(t, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, Upgrade.StoreUpgrades.Renamed)
}
