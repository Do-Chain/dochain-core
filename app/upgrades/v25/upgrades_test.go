package v25

import (
	"testing"

	valuefeetypes "github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	"github.com/stretchr/testify/require"
)

func TestMainnetV25ParamsAddFeeExemptAddresses(t *testing.T) {
	v24 := valuefeetypes.MainnetV24Params()
	v25 := valuefeetypes.MainnetV25Params()

	require.Equal(t, uint32(25), v25.PolicyVersion)
	require.Equal(t, v24.RateBps, v25.RateBps)
	require.Equal(t, v24.MinFee, v25.MinFee)
	require.Equal(t, v24.DenomValueRates, v25.DenomValueRates)
	require.Len(t, v24.FeeExemptAddresses, 5)
	require.Len(t, v25.FeeExemptAddresses, 9)
	require.Contains(t, v25.FeeExemptAddresses, "do1mjgpy7wjhelpxssm8gl63fz5crl4tydvc2g5pj")
	require.Contains(t, v25.FeeExemptAddresses, "do1xas7z5ldgd296gj37ux2cwc37lt87h008v090d")
	require.Contains(t, v25.FeeExemptAddresses, "do1ftxpqe0dj6rcayz3fhw3xvesq47a26es8enyuu")
	require.Contains(t, v25.FeeExemptAddresses, "do12emm6cvg4sqh9cxsd6ys42kq67gyfjnw3jhhjr")
	require.Contains(t, v25.FeeExemptAddresses, "do1fulr5u0saspce2zuh2quqppesfx65cg5cu9eaz")
	require.Contains(t, v25.FeeExemptAddresses, "do16w707l5t2ru9xuhjguc2zcf59845j0urt5c0r0")
	require.Contains(t, v25.FeeExemptAddresses, "do1whutlz9ddrnmjx686vpqexsng8sttdzvsevw3u")
	require.Contains(t, v25.FeeExemptAddresses, "do10zjdun4e8pc8zxcj2j9q96ra4jzld6cvp47tq2")
	require.Contains(t, v25.FeeExemptAddresses, "do100jvak5pcrk9jvfwl0n3stnzfhjv92kndlrlaz")
	require.Len(t, v25.ParamSetPairs(), len(v24.ParamSetPairs()))
	require.Len(t, valuefeetypes.MainnetFeeExemptAddresses(24), 5)
	require.Len(t, valuefeetypes.MainnetFeeExemptAddresses(25), 9)
	require.NoError(t, v25.Validate())
}
