package ante

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAllowedMFAMemo(t *testing.T) {
	longSignature := strings.Repeat("a", 512)

	require.False(t, isAllowedMFAMemo(strings.Repeat("x", 345), 345))
	require.True(t, isAllowedMFAMemo(`{"dochain_mfa":{"approvals":[{"signature":"`+longSignature+`"}]}}`, 345))
	require.False(t, isAllowedMFAMemo(`{"dochain_mfa":{}}`, int(maxMFAMemoCharacters)+1))
}
