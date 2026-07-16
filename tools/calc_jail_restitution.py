#!/usr/bin/env python3
"""Calculate apparent DO losses around validator jail/slash events.

The public LCD may not retain historical state. If a pre-jail height has been
pruned, run this against an archive LCD or a restored node that has that height.
"""

from __future__ import annotations

import argparse
import csv
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from collections import defaultdict
from dataclasses import dataclass
from decimal import Decimal, getcontext
from pathlib import Path


getcontext().prec = 80

DEFAULT_LCD = "https://www.do-chain.com/lcd"
DEFAULT_VALIDATORS = [
    ("dovaloper1rr5jz7wt4fyckn674ad02u68crh95ysd2s4sl0", 503_999),
    ("dovaloper1u6aznk36sxwgfd0hyq4jgmlgufpsv3mh3lkudk", 508_033),
]


@dataclass(frozen=True)
class DelegationBalance:
    delegator: str
    shares: Decimal
    amount: int


def request_json(lcd: str, path: str, height: int | None = None) -> dict:
    url = lcd.rstrip("/") + path
    req = urllib.request.Request(url)
    if height is not None:
        req.add_header("x-cosmos-block-height", str(height))

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{exc.code} from {url}: {body}") from exc


def fetch_delegations(lcd: str, validator: str, height: int | None) -> dict[str, DelegationBalance]:
    balances: dict[str, DelegationBalance] = {}
    next_key = ""
    while True:
        query = {"pagination.limit": "1000"}
        if next_key:
            query["pagination.key"] = next_key
        path = (
            f"/cosmos/staking/v1beta1/validators/{validator}/delegations?"
            + urllib.parse.urlencode(query)
        )
        data = request_json(lcd, path, height)
        for item in data.get("delegation_responses", []):
            delegation = item["delegation"]
            balance = item["balance"]
            delegator = delegation["delegator_address"]
            balances[delegator] = DelegationBalance(
                delegator=delegator,
                shares=Decimal(delegation["shares"]),
                amount=int(balance["amount"]),
            )

        next_key = data.get("pagination", {}).get("next_key") or ""
        if not next_key:
            return balances


def parse_validator_spec(value: str) -> tuple[str, int]:
    if "@" not in value:
        raise argparse.ArgumentTypeError("validator spec must be dovaloper...@height")
    validator, height = value.rsplit("@", 1)
    try:
        return validator, int(height.replace("_", ""))
    except ValueError as exc:
        raise argparse.ArgumentTypeError(f"invalid height in {value!r}") from exc


def micro_to_do(amount: int) -> str:
    return f"{Decimal(amount) / Decimal(1_000_000):f}"


def write_rows(path: Path, rows: list[dict[str, str]]) -> None:
    if not rows:
        path.write_text("", encoding="utf-8")
        return
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0].keys()))
        writer.writeheader()
        writer.writerows(rows)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--lcd", default=DEFAULT_LCD, help=f"LCD base URL, default {DEFAULT_LCD}")
    parser.add_argument(
        "--validator",
        action="append",
        type=parse_validator_spec,
        help="Validator and pre-jail height as dovaloper...@height. Can be repeated.",
    )
    parser.add_argument("--current-height", type=int, default=None, help="Post-jail height. Defaults to latest.")
    parser.add_argument("--rows-csv", default="jail_restitution_rows.csv", help="Per-validator CSV output.")
    parser.add_argument("--summary-csv", default="jail_restitution_summary.csv", help="Per-delegator CSV output.")
    args = parser.parse_args()

    validators = args.validator or DEFAULT_VALIDATORS
    rows: list[dict[str, str]] = []
    totals: dict[str, int] = defaultdict(int)

    for validator, pre_height in validators:
        try:
            pre = fetch_delegations(args.lcd, validator, pre_height)
            current = fetch_delegations(args.lcd, validator, args.current_height)
        except RuntimeError as err:
            print(f"Cannot query {validator} at height {pre_height}: {err}", file=sys.stderr)
            print("Use an archive LCD or a node restored with that historical state.", file=sys.stderr)
            return 2

        for delegator, pre_balance in sorted(pre.items()):
            current_balance = current.get(delegator)
            current_amount = current_balance.amount if current_balance else 0
            lost = max(pre_balance.amount - current_amount, 0)
            if lost == 0:
                continue

            totals[delegator] += lost
            rows.append(
                {
                    "validator_address": validator,
                    "pre_height": str(pre_height),
                    "current_height": str(args.current_height or "latest"),
                    "delegator_address": delegator,
                    "pre_shares": f"{pre_balance.shares:f}",
                    "current_shares": f"{current_balance.shares:f}" if current_balance else "0",
                    "pre_amount_udo": str(pre_balance.amount),
                    "current_amount_udo": str(current_amount),
                    "apparent_loss_udo": str(lost),
                    "apparent_loss_do": micro_to_do(lost),
                    "review_note": "review voluntary unbonding/redelegation before payment",
                }
            )

    summary_rows = [
        {
            "delegator_address": delegator,
            "apparent_loss_udo": str(amount),
            "apparent_loss_do": micro_to_do(amount),
            "review_note": "pay from treasury only after manual review",
        }
        for delegator, amount in sorted(totals.items())
    ]

    rows_path = Path(args.rows_csv)
    summary_path = Path(args.summary_csv)
    write_rows(rows_path, rows)
    write_rows(summary_path, summary_rows)

    total_udo = sum(totals.values())
    print(f"Wrote {len(rows)} detailed rows to {rows_path}")
    print(f"Wrote {len(summary_rows)} delegator summary rows to {summary_path}")
    print(f"Total apparent restitution: {total_udo}udo ({micro_to_do(total_udo)} DO)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
