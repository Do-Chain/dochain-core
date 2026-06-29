#!/usr/bin/env python3
import argparse
import json
import sys
import urllib.error
import urllib.request


ACCESS_TYPES = {
    0: ("Unspecified", "undefined/placeholder"),
    1: ("Nobody", "no one is allowed"),
    2: ("OnlyAddress", "only one specific address is allowed; deprecated in newer wasmd"),
    3: ("Everybody", "anyone is allowed"),
    4: ("AnyOfAddresses", "any address in the allowlist is allowed"),
}

ACCESS_NAME_TO_NUMBER = {
    "unspecified": 0,
    "access_type_unspecified": 0,
    "nobody": 1,
    "access_type_nobody": 1,
    "onlyaddress": 2,
    "only_address": 2,
    "access_type_only_address": 2,
    "everybody": 3,
    "access_type_everybody": 3,
    "anyofaddresses": 4,
    "any_of_addresses": 4,
    "access_type_any_of_addresses": 4,
}

PARAM_ENDPOINTS = (
    "/cosmwasm/wasm/v1/codes/params",
    "/cosmwasm/wasm/v1/params",
)


def http_json(url, timeout):
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=timeout) as response:
        return json.load(response)


def query_params(lcd, timeout):
    base = lcd.rstrip("/")
    errors = []
    for endpoint in PARAM_ENDPOINTS:
        url = f"{base}{endpoint}"
        try:
            return endpoint, http_json(url, timeout)
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace").strip()
            errors.append(f"{endpoint}: HTTP {exc.code} {body}")
        except urllib.error.URLError as exc:
            errors.append(f"{endpoint}: {exc.reason}")

    raise RuntimeError("unable to query wasm params:\n  " + "\n  ".join(errors))


def access_number(value):
    if isinstance(value, int):
        return value
    if isinstance(value, str):
        stripped = value.strip()
        if stripped.isdigit():
            return int(stripped)
        return ACCESS_NAME_TO_NUMBER.get(stripped.lower())
    return None


def access_info(value):
    number = access_number(value)
    if number in ACCESS_TYPES:
        name, meaning = ACCESS_TYPES[number]
        return number, name, meaning
    return number, str(value), "unknown access type"


def print_access(label, value, addresses=None):
    number, name, meaning = access_info(value)
    number_text = "unknown" if number is None else str(number)
    print(f"{label}: {number_text} = {name} - {meaning}")
    if addresses is not None:
        if addresses:
            print("Allowed upload addresses:")
            for address in addresses:
                print(f"  - {address}")
        else:
            print("Allowed upload addresses: none")


def main():
    parser = argparse.ArgumentParser(
        description="Query CosmWasm code upload access params from a chain LCD."
    )
    parser.add_argument(
        "--lcd",
        default="https://www.do-chain.com/lcd",
        help="LCD base URL, default: https://www.do-chain.com/lcd",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=20,
        help="HTTP timeout in seconds, default: 20",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Print the raw params JSON response.",
    )
    args = parser.parse_args()

    try:
        endpoint, data = query_params(args.lcd, args.timeout)
    except RuntimeError as exc:
        print(exc, file=sys.stderr)
        return 1

    if args.json:
        print(json.dumps(data, indent=2, sort_keys=True))
        return 0

    params = data.get("params") or {}
    upload = params.get("code_upload_access") or {}
    upload_permission = upload.get("permission", upload.get("access_type"))
    upload_addresses = upload.get("addresses")
    if upload_addresses is None:
        legacy_address = upload.get("address")
        upload_addresses = [legacy_address] if legacy_address else []

    print(f"LCD: {args.lcd.rstrip('/')}")
    print(f"Endpoint: {endpoint}")
    print_access("Code upload access", upload_permission, upload_addresses)
    print_access(
        "Instantiate default permission",
        params.get("instantiate_default_permission"),
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
