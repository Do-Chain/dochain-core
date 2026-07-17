package util

import "testing"

func TestDoChainHDPathDefaults(t *testing.T) {
	if CoinType != 888 {
		t.Fatalf("expected DoChain coin type 888, got %d", CoinType)
	}

	if Purpose != 44 {
		t.Fatalf("expected BIP-44 purpose 44, got %d", Purpose)
	}
}
