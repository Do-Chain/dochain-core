package app

import (
	"encoding/json"
)

const (
	mainnetZeroTobinTax = "0.000000000000000000"
	mainnetUSDTobinTax  = "0.002500000000000000"
)

// GenesisState The genesis state of the blockchain is represented here as a map of raw json
// messages key'd by a identifier string.
// The identifier is used to determine which module genesis information belongs
// to so it may be appropriately routed during init chain.
// Within this application default genesis information is retrieved from
// the ModuleBasicManager which populates json from each BasicModule
// object provided to it during init.
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState() GenesisState {
	encCfg := MakeEncodingConfig()
	genesis := ModuleBasics.DefaultGenesis(encCfg.Marshaler)
	HardenMainnetGenesisDefaults(genesis)
	return genesis
}

// HardenMainnetGenesisDefaults applies conservative launch defaults to generated
// genesis state. It intentionally touches only genesis-facing params.
func HardenMainnetGenesisDefaults(genesis map[string]json.RawMessage) {
	hardenWasmGenesis(genesis)
	hardenIBCGenesis(genesis)
	hardenICAGenesis(genesis)
	hardenTransferGenesis(genesis)
	hardenOracleGenesis(genesis)
	hardenBankGenesis(genesis)
}

func hardenWasmGenesis(genesis map[string]json.RawMessage) {
	patchGenesisModule(genesis, "wasm", func(state map[string]any) {
		params := ensureObject(state, "params")
		params["code_upload_access"] = map[string]any{
			"permission": "Nobody",
			"addresses":  []any{},
		}
		params["instantiate_default_permission"] = "Nobody"
	})
}

func hardenIBCGenesis(genesis map[string]json.RawMessage) {
	patchGenesisModule(genesis, "ibc", func(state map[string]any) {
		clientGenesis := ensureObject(state, "client_genesis")
		params := ensureObject(clientGenesis, "params")
		params["allowed_clients"] = []any{"07-tendermint"}
	})
}

func hardenICAGenesis(genesis map[string]json.RawMessage) {
	patchGenesisModule(genesis, "interchainaccounts", func(state map[string]any) {
		controller := ensureObject(state, "controller_genesis_state")
		controllerParams := ensureObject(controller, "params")
		controllerParams["controller_enabled"] = false

		host := ensureObject(state, "host_genesis_state")
		hostParams := ensureObject(host, "params")
		hostParams["host_enabled"] = false
		hostParams["allow_messages"] = []any{}
	})
}

func hardenTransferGenesis(genesis map[string]json.RawMessage) {
	patchGenesisModule(genesis, "transfer", func(state map[string]any) {
		params := ensureObject(state, "params")
		params["send_enabled"] = false
		params["receive_enabled"] = false
	})
}

func hardenOracleGenesis(genesis map[string]json.RawMessage) {
	patchGenesisModule(genesis, "oracle", func(state map[string]any) {
		params := ensureObject(state, "params")
		params["whitelist"] = []any{
			map[string]any{"name": "udo", "tobin_tax": mainnetZeroTobinTax},
			map[string]any{"name": "uusd", "tobin_tax": mainnetUSDTobinTax},
		}
		state["exchange_rates"] = []any{}
		state["tobin_taxes"] = []any{}
	})
}

func hardenBankGenesis(genesis map[string]json.RawMessage) {
	patchGenesisModule(genesis, "bank", func(state map[string]any) {
		metadata, _ := state["denom_metadata"].([]any)
		metadata = ensureMetadata(metadata, denomMetadata("udo", "do", "Do", "DO"))
		metadata = ensureMetadata(metadata, denomMetadata("udodx", "dodx", "Do governance token", "DODx"))
		metadata = ensureMetadata(metadata, denomMetadata("uusd", "usd", "Do USD oracle unit", "USD"))
		if bankHasDenom(state, "ubaked") {
			metadata = ensureMetadata(metadata, denomMetadata("ubaked", "baked", "Do baked token", "BAKED"))
		}
		state["denom_metadata"] = metadata
	})
}

func patchGenesisModule(genesis map[string]json.RawMessage, moduleName string, patch func(map[string]any)) {
	raw, ok := genesis[moduleName]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return
	}

	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		return
	}

	patch(state)

	if bz, err := json.Marshal(state); err == nil {
		genesis[moduleName] = bz
	}
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}

	value := map[string]any{}
	parent[key] = value
	return value
}

func denomMetadata(base, display, name, symbol string) map[string]any {
	return map[string]any{
		"description": name,
		"denom_units": []any{
			map[string]any{"denom": base, "exponent": float64(0), "aliases": []any{"micro" + display}},
			map[string]any{"denom": display, "exponent": float64(6), "aliases": []any{}},
		},
		"base":     base,
		"display":  display,
		"name":     name,
		"symbol":   symbol,
		"uri":      "",
		"uri_hash": "",
	}
}

func ensureMetadata(existing []any, metadata map[string]any) []any {
	base, _ := metadata["base"].(string)
	for i, item := range existing {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if entryBase, _ := entry["base"].(string); entryBase == base {
			existing[i] = metadata
			return existing
		}
	}
	return append(existing, metadata)
}

func bankHasDenom(bank map[string]any, denom string) bool {
	if coins, ok := bank["supply"].([]any); ok && coinListHasDenom(coins, denom) {
		return true
	}

	if balances, ok := bank["balances"].([]any); ok {
		for _, item := range balances {
			balance, ok := item.(map[string]any)
			if !ok {
				continue
			}
			coins, _ := balance["coins"].([]any)
			if coinListHasDenom(coins, denom) {
				return true
			}
		}
	}

	return false
}

func coinListHasDenom(coins []any, denom string) bool {
	for _, item := range coins {
		coin, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if coinDenom, _ := coin["denom"].(string); coinDenom == denom {
			return true
		}
	}
	return false
}
