package app_test

import (
	"fmt"
	"testing"

	"github.com/shamaton/msgpack/v2"
	"github.com/stretchr/testify/require"
)

// Regression coverage for GO-2026-4740: truncated fixext values must return an
// error rather than panic while the vulnerability database has no fixed
// version newer than msgpack/v2 v2.4.1.
func TestMsgpackTruncatedFixExtDoesNotPanic(t *testing.T) {
	for code := byte(0xd4); code <= 0xd8; code++ {
		for _, input := range [][]byte{{code}, {code, 0}} {
			input := input
			t.Run(fmt.Sprintf("%x-%d-bytes", code, len(input)), func(t *testing.T) {
				var decoded any
				require.Error(t, msgpack.Unmarshal(input, &decoded))
			})
		}
	}
}
