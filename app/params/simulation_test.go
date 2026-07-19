package params

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeSimulationTxConfigProvidesAddressCodecs(t *testing.T) {
	txConfig := MakeSimulationTxConfig()
	addressBytes := bytes.Repeat([]byte{1}, 20)

	for _, addressCodec := range []interface {
		BytesToString([]byte) (string, error)
		StringToBytes(string) ([]byte, error)
	}{
		txConfig.SigningContext().AddressCodec(),
		txConfig.SigningContext().ValidatorAddressCodec(),
	} {
		address, err := addressCodec.BytesToString(addressBytes)
		require.NoError(t, err)
		require.NotEmpty(t, address)

		decoded, err := addressCodec.StringToBytes(address)
		require.NoError(t, err)
		require.Equal(t, addressBytes, decoded)
	}
}
