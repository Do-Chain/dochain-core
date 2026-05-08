#!/usr/bin/env python3
"""Apply DoChain mainnet-candidate hardening to a genesis JSON file."""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

ZERO_TOBIN_TAX = "0.000000000000000000"
USD_TOBIN_TAX = "0.002500000000000000"


def metadata(base: str, display: str, name: str, symbol: str) -> dict[str, Any]:
    return {
        "description": name,
        "denom_units": [
            {"denom": base, "exponent": 0, "aliases": [f"micro{display}"]},
            {"denom": display, "exponent": 6, "aliases": []},
        ],
        "base": base,
        "display": display,
        "name": name,
        "symbol": symbol,
        "uri": "",
        "uri_hash": "",
    }


def upsert_metadata(items: list[dict[str, Any]], item: dict[str, Any]) -> None:
    for index, existing in enumerate(items):
        if existing.get("base") == item["base"]:
            items[index] = item
            return
    items.append(item)


def coin_list_has_denom(coins: list[dict[str, Any]], denom: str) -> bool:
    return any(coin.get("denom") == denom for coin in coins)


def bank_has_denom(bank: dict[str, Any], denom: str) -> bool:
    if coin_list_has_denom(bank.get("supply", []), denom):
        return True
    return any(coin_list_has_denom(balance.get("coins", []), denom) for balance in bank.get("balances", []))


def ensure_dict(parent: dict[str, Any], key: str) -> dict[str, Any]:
    value = parent.get(key)
    if not isinstance(value, dict):
        value = {}
        parent[key] = value
    return value


def ensure_list(parent: dict[str, Any], key: str) -> list[Any]:
    value = parent.get(key)
    if not isinstance(value, list):
        value = []
        parent[key] = value
    return value


def harden_genesis(genesis: dict[str, Any]) -> list[str]:
    warnings: list[str] = []
    app_state = ensure_dict(genesis, "app_state")

    wasm = ensure_dict(app_state, "wasm")
    wasm_params = ensure_dict(wasm, "params")
    wasm_params["code_upload_access"] = {"permission": "Nobody", "addresses": []}
    wasm_params["instantiate_default_permission"] = "Nobody"

    ibc = ensure_dict(app_state, "ibc")
    ensure_dict(ensure_dict(ibc, "client_genesis"), "params")["allowed_clients"] = ["07-tendermint"]

    ica = ensure_dict(app_state, "interchainaccounts")
    ensure_dict(ensure_dict(ica, "controller_genesis_state"), "params")["controller_enabled"] = False
    host_params = ensure_dict(ensure_dict(ica, "host_genesis_state"), "params")
    host_params["host_enabled"] = False
    host_params["allow_messages"] = []

    transfer = ensure_dict(app_state, "transfer")
    transfer_params = ensure_dict(transfer, "params")
    transfer_params["send_enabled"] = False
    transfer_params["receive_enabled"] = False

    oracle = ensure_dict(app_state, "oracle")
    oracle_params = ensure_dict(oracle, "params")
    oracle_params["whitelist"] = [
        {"name": "udo", "tobin_tax": ZERO_TOBIN_TAX},
        {"name": "uusd", "tobin_tax": USD_TOBIN_TAX},
    ]
    oracle["exchange_rates"] = []
    oracle["tobin_taxes"] = []

    bank = ensure_dict(app_state, "bank")
    denom_metadata = ensure_list(bank, "denom_metadata")
    upsert_metadata(denom_metadata, metadata("udo", "do", "Do", "DO"))
    upsert_metadata(denom_metadata, metadata("uusd", "usd", "Do USD oracle unit", "USD"))
    if bank_has_denom(bank, "ubaked"):
        upsert_metadata(denom_metadata, metadata("ubaked", "baked", "Do baked token", "BAKED"))

    gen_txs = ensure_dict(app_state, "genutil").get("gen_txs", [])
    if len(gen_txs) < 4:
        warnings.append(f"genesis has {len(gen_txs)} validator gentx(s); mainnet target is at least 4")

    if ensure_dict(app_state, "auth").get("accounts", []):
        warnings.append("document foundation and launch allocation ownership before audit submission")

    return warnings


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path, help="source genesis.json")
    parser.add_argument("output", type=Path, help="destination hardened genesis.json")
    parser.add_argument(
        "--genesis-time",
        help='optional RFC3339/RFC3339Nano timestamp, or "now" to set current UTC time',
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="exit non-zero when mainnet-readiness warnings remain",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    genesis = json.loads(args.input.read_text(encoding="utf-8-sig"))

    warnings = harden_genesis(genesis)

    if args.genesis_time == "now":
        genesis["genesis_time"] = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
    elif args.genesis_time:
        genesis["genesis_time"] = args.genesis_time

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(genesis, indent=2, sort_keys=False) + "\n", encoding="utf-8")

    for warning in warnings:
        print(f"WARNING: {warning}", file=sys.stderr)

    return 1 if args.strict and warnings else 0


if __name__ == "__main__":
    raise SystemExit(main())
