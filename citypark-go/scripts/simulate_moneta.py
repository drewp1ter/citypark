#!/usr/bin/env python3
"""Simulate a MONETA Pay URL callback using only Python's standard library."""

import argparse
import hashlib
import hmac
import os
from pathlib import Path
import re
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import HTTPRedirectHandler, Request, build_opener
import xml.etree.ElementTree as ET


def load_settings(path):
    """Read literal KEY=value entries; environment variables take precedence."""
    settings = {}
    if path.exists():
        for line in path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            key, separator, value = line.removeprefix("export ").partition("=")
            if not separator:
                raise ValueError("Expected KEY=value in env file")
            value = value.strip()
            if len(value) >= 2 and value[0] == value[-1] and value[0] in "'\"":
                value = value[1:-1]
            settings[key.strip()] = value
    settings.update(os.environ)
    return settings


def digest(text):
    # MD5 is required by MONETA's protocol.
    return hashlib.md5(text.encode("utf-8"), usedforsecurity=False).hexdigest()


def signed_fields(order_id, merchant, secret, mode, operation="", amount="", command=""):
    fields = {
        "MNT_ID": merchant,
        "MNT_TRANSACTION_ID": str(order_id),
        "MNT_CURRENCY_CODE": "RUB",
        "MNT_TEST_MODE": mode,
    }
    if operation:
        fields["MNT_OPERATION_ID"] = operation
    if amount:
        fields["MNT_AMOUNT"] = amount
    if command:
        fields["MNT_COMMAND"] = command
    fields["MNT_SIGNATURE"] = digest(
        command + merchant + str(order_id) + operation + amount + "RUB" + mode + secret
    )
    return fields


def post(url, fields):
    request = Request(url, data=urlencode(fields).encode("ascii"), method="POST")
    request.add_header("Content-Type", "application/x-www-form-urlencoded")
    # Never follow redirects carrying a signed payment request.
    class NoRedirect(HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, headers, newurl):
            return None

    try:
        with build_opener(NoRedirect).open(request, timeout=15) as response:
            return response.read().decode("utf-8")
    except HTTPError as error:
        raise ValueError(f"Webhook returned HTTP {error.code}: {error.read().decode('utf-8', errors='replace')}") from error


def checked_amount(body, merchant, order_id, secret):
    root = ET.fromstring(body)
    code = root.findtext("MNT_RESULT_CODE", "")
    signature = root.findtext("MNT_SIGNATURE", "")
    expected = digest(code + merchant + str(order_id) + secret)
    if (
        root.tag != "MNT_RESPONSE"
        or root.findtext("MNT_ID") != merchant
        or root.findtext("MNT_TRANSACTION_ID") != str(order_id)
        or not hmac.compare_digest(signature.lower(), expected)
    ):
        raise ValueError("Invalid signed Check URL response")
    if code == "200":
        raise ValueError("Order is already paid")
    if code not in ("100", "402"):
        raise ValueError(f"Order is not payable (MONETA result {code})")
    amount = root.findtext("MNT_AMOUNT", "")
    if not re.fullmatch(r"[0-9]+\.[0-9]{2}", amount):
        raise ValueError("Check URL did not return a valid two-decimal amount")
    return amount


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("order_id", type=int, help="Order number, e.g. 16")
    parser.add_argument("--base-url", default="http://localhost:3000", help="Backend origin (default: %(default)s)")
    parser.add_argument("--env-file", type=Path, default=Path(__file__).resolve().parents[1] / ".env")
    parser.add_argument("--amount", help="Rubles, e.g. 790.00; bypass Check URL")
    parser.add_argument("--operation-id", help="Simulated MONETA operation ID (default: simulated-ORDER)")
    parser.add_argument("--dry-run", action="store_true", help="Print order and amount without sending Pay URL; may call Check URL")
    args = parser.parse_args(argv)
    try:
        if args.order_id < 1:
            raise ValueError("Order number must be positive")
        settings = load_settings(args.env_file)
        merchant, secret = settings.get("MONETA_ID", ""), settings.get("MONETA_SECRET", "")
        if not merchant or not secret:
            raise ValueError("Set MONETA_ID and MONETA_SECRET in the env file or environment")
        mode = "1" if settings.get("MONETA_TEST_MODE", "false") == "true" else "0"
        base = args.base_url.rstrip("/") + "/api/v1/webhooks/moneta"
        amount = args.amount
        if amount is None:
            check = signed_fields(args.order_id, merchant, secret, mode, command="CHECK")
            amount = checked_amount(post(base + "/check", check), merchant, args.order_id, secret)
        if not re.fullmatch(r"[0-9]+\.[0-9]{2}", amount):
            raise ValueError("Amount must use two decimals, e.g. 790.00")
        print(f"Order {args.order_id}: {amount} RUB; MNT_TEST_MODE={mode}")
        if args.dry_run:
            print("Dry run: Pay URL notification not sent.")
            return 0
        operation = args.operation_id or f"simulated-{args.order_id}"
        response = post(base, signed_fields(args.order_id, merchant, secret, mode, operation, amount))
        if response != "SUCCESS":
            raise ValueError(f"Unexpected Pay URL response: {response!r}")
        print("SUCCESS")
        return 0
    except (ValueError, OSError, URLError, ET.ParseError) as error:
        print(f"Error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
