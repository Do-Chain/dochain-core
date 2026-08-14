package v24

import (
	"testing"

	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	"github.com/stretchr/testify/require"
)

func TestMainnetV24ParamsAddFeeExemptAddresses(t *testing.T) {
	v23 := valuefeetypes.MainnetV23Params()
	v24 := valuefeetypes.MainnetV24Params()

	require.Equal(t, uint32(24), v24.PolicyVersion)
	require.Equal(t, v23.RateBps, v24.RateBps)
	require.Equal(t, v23.MinFee, v24.MinFee)
	require.Equal(t, v23.DenomValueRates, v24.DenomValueRates)
	require.Len(t, v24.FeeExemptAddresses, 5)
	require.Contains(t, v24.FeeExemptAddresses, "do1mjgpy7wjhelpxssm8gl63fz5crl4tydvc2g5pj")
	require.Contains(t, v24.FeeExemptAddresses, "do1xas7z5ldgd296gj37ux2cwc37lt87h008v090d")
	require.Contains(t, v24.FeeExemptAddresses, "do1ftxpqe0dj6rcayz3fhw3xvesq47a26es8enyuu")
	require.Contains(t, v24.FeeExemptAddresses, "do12emm6cvg4sqh9cxsd6ys42kq67gyfjnw3jhhjr")
	require.Contains(t, v24.FeeExemptAddresses, "do1fulr5u0saspce2zuh2quqppesfx65cg5cu9eaz")
	require.Len(t, v24.ParamSetPairs(), len(v23.ParamSetPairs()))
	require.NoError(t, v24.Validate())
}
