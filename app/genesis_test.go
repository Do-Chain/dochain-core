package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHardenMainnetGenesisDefaultsDisablesDelegatorWideSlashing(t *testing.T) {
	genesis := map[string]json.RawMessage{
		"slashing": json.RawMessage(`{"params":{"slash_fraction_double_sign":"0.050000000000000000","slash_fraction_downtime":"0.010000000000000000"}}`),
	}

	HardenMainnetGenesisDefaults(genesis)

	var slashing map[string]any
	require.NoError(t, json.Unmarshal(genesis["slashing"], &slashing))
	params := slashing["params"].(map[string]any)
	require.Equal(t, mainnetZeroTobinTax, params["slash_fraction_double_sign"])
	require.Equal(t, mainnetZeroTobinTax, params["slash_fraction_downtime"])
}
