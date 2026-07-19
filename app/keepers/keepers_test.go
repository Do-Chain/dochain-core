package keepers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWasmCapabilitiesPreserveCustomBindings(t *testing.T) {
	capabilities := wasmCapabilities()

	require.Contains(t, capabilities, doWasmCapability)
	require.Contains(t, capabilities, legacyTerraWasmCapability)
}
