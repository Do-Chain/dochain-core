package v23

import (
	"testing"

	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	"github.com/stretchr/testify/require"
)

func TestMainnetV23ParamsKeepExistingSchema(t *testing.T) {
	v22 := valuefeetypes.MainnetV22Params()
	v23 := valuefeetypes.MainnetV23Params()

	require.Equal(t, uint32(22), v22.PolicyVersion)
	require.Equal(t, uint32(23), v23.PolicyVersion)
	v22.PolicyVersion = 23
	require.Equal(t, v22, v23)
	v22Pairs := valuefeetypes.MainnetV22Params()
	require.Len(t, v23.ParamSetPairs(), len(v22Pairs.ParamSetPairs()))
	require.NoError(t, v23.Validate())
}
