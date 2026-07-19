package wasmbinding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyStargateAliasesResolveOnlyToWhitelistedDoQueries(t *testing.T) {
	for legacyPath, canonicalPath := range legacyStargateQueryAliases {
		t.Run(legacyPath, func(t *testing.T) {
			require.Equal(t, canonicalPath, canonicalStargateQueryPath(legacyPath))
			_, err := GetWhitelistedQuery(legacyPath)
			require.NoError(t, err)
		})
	}
}

func TestUnknownStargateQueryRemainsBlocked(t *testing.T) {
	const queryPath = "/terra.unknown.v1.Query/Secrets"

	require.Equal(t, queryPath, canonicalStargateQueryPath(queryPath))
	_, err := GetWhitelistedQuery(queryPath)
	require.Error(t, err)
}
