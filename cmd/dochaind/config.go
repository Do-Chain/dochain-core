package main

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

const (
	// DefaultIAVLCacheSize defines the number of iavl cache items.
	// Raised further for higher-throughput infrastructure.
	// This assumes you control the validator hardware and can afford the memory.
	DefaultIAVLCacheSize = 3_125_000

	// For better performance on modern nodes, keep fast node enabled by default.
	IavlDisablefastNodeDefault = false

	// DefaultWasmMemoryCacheSize is in MiB.
	// Raised for heavier smart contract usage and better cache hit rate.
	DefaultWasmMemoryCacheSize = 1024

	// DefaultWasmSmartQueryGasLimit raises the ceiling for smart queries.
	DefaultWasmSmartQueryGasLimit = 5_000_000
)

// DoAppConfig specifies app config.
type DoAppConfig struct {
	serverconfig.Config
	Wasm wasmtypes.NodeConfig `mapstructure:"wasm"`
}

// WasmConfigTemplate toml snippet for app.toml
func WasmConfigTemplate(c wasmtypes.NodeConfig) string {
	simGasLimit := `# simulation_gas_limit =`
	if c.SimulationGasLimit != nil {
		simGasLimit = fmt.Sprintf(`simulation_gas_limit = %d`, *c.SimulationGasLimit)
	}

	return fmt.Sprintf(`

###############################################################################
###                                  WASM                                   ###
###############################################################################

[wasm]
# Smart query gas limit is the max gas to be used in a smart query contract call
query_gas_limit = %d

# in-memory cache for Wasm contracts. Set to 0 to disable.
# The value is in MiB not bytes
memory_cache_size = %d

# Simulation gas limit is the max gas to be used in a tx simulation call.
# When not set the consensus max block gas is used instead
%s
`, c.SmartQueryGasLimit, c.MemoryCacheSize, simGasLimit)
}

// DefaultWasmConfigTemplate toml snippet with default values for app.toml
func DefaultWasmConfigTemplate() string {
	return WasmConfigTemplate(wasmtypes.DefaultNodeConfig())
}

// initAppConfig helps to override default app.toml template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// Start from the SDK default server config.
	srvCfg := serverconfig.DefaultConfig()

	// Performance-oriented defaults for freshly initialized nodes.
	srvCfg.IAVLCacheSize = DefaultIAVLCacheSize
	srvCfg.IAVLDisableFastNode = IavlDisablefastNodeDefault

	// The SDK default minimum gas price is empty, which can cause startup halt
	// if validators do not set it manually. Keep a chain-safe default here.
	srvCfg.MinGasPrices = "0udo"

	// Pruning defaults
	// Pruning defaults
srvCfg.Pruning = "custom"
srvCfg.PruningKeepRecent = "50"

srvCfg.PruningInterval = "10"



	// API defaults
	srvCfg.API.Enable = true
	srvCfg.API.Swagger = true
	srvCfg.API.Address = "tcp://0.0.0.0:1317"

	// gRPC defaults
	srvCfg.GRPC.Enable = true
	srvCfg.GRPC.Address = "localhost:9090"

	// gRPC-web defaults
	srvCfg.GRPCWeb.Enable = true

	// App-side mempool defaults
	srvCfg.Mempool.MaxTxs = 20000

	// Start from the default Wasm config and raise the useful defaults.
	wasmCfg := wasmtypes.DefaultNodeConfig()
	wasmCfg.MemoryCacheSize = DefaultWasmMemoryCacheSize
	wasmCfg.SmartQueryGasLimit = DefaultWasmSmartQueryGasLimit

	doAppConfig := DoAppConfig{
		Config: *srvCfg,
		Wasm:   wasmCfg,
	}

	doAppTemplate := serverconfig.DefaultConfigTemplate + WasmConfigTemplate(wasmCfg)

	return doAppTemplate, doAppConfig
}