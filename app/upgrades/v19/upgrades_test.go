package v19

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	validatorrewardstypes "github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount("do", "dopub")
}

func TestWhitelistDodexWasmUploaderFromNobody(t *testing.T) {
	params := wasmtypes.DefaultParams()
	params.CodeUploadAccess = wasmtypes.AccessConfig{Permission: wasmtypes.AccessTypeNobody}
	params.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody

	updated := whitelistDodexWasmUploader(params)

	require.Equal(t, wasmtypes.AccessTypeAnyOfAddresses, updated.CodeUploadAccess.Permission)
	require.Equal(t, []string{dodexWasmUploadAdmin}, updated.CodeUploadAccess.Addresses)
	require.Equal(t, wasmtypes.AccessTypeNobody, updated.InstantiateDefaultPermission)
	require.NoError(t, updated.ValidateBasic())
}

func TestWhitelistDodexWasmUploaderPreservesExistingAllowlist(t *testing.T) {
	existing := "do1f2ywc5wxpxrh27rdtt0t4alkal7320n6z6fqcu"
	params := wasmtypes.DefaultParams()
	params.CodeUploadAccess = wasmtypes.AccessConfig{
		Permission: wasmtypes.AccessTypeAnyOfAddresses,
		Addresses:  []string{existing, dodexWasmUploadAdmin},
	}
	params.InstantiateDefaultPermission = wasmtypes.AccessTypeNobody

	updated := whitelistDodexWasmUploader(params)

	require.Equal(t, wasmtypes.AccessTypeAnyOfAddresses, updated.CodeUploadAccess.Permission)
	require.Equal(t, []string{existing, dodexWasmUploadAdmin}, updated.CodeUploadAccess.Addresses)
	require.Equal(t, wasmtypes.AccessTypeNobody, updated.InstantiateDefaultPermission)
	require.NoError(t, updated.ValidateBasic())
}

func TestV19AddsValidatorRewardsStore(t *testing.T) {
	require.Equal(t, []string{validatorrewardstypes.StoreKey}, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, Upgrade.StoreUpgrades.Renamed)
}
