package params

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/std"
)

// MakeSimulationTxConfig constructs the transaction config used by module
// simulations, including the address codecs required by protobuf signers.
func MakeSimulationTxConfig() client.TxConfig {
	encodingConfig := MakeEncodingConfig()
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	return encodingConfig.TxConfig
}
