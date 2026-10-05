"""Generate signing golden vectors with the official Hyperliquid Python SDK.

Usage (from the repository root):

    pip install hyperliquid-python-sdk
    python testdata/gen_signing_vectors.py > testdata/signing_vectors.json

Every vector is produced by the SDK's own signing functions, so the Go tests
prove byte-for-byte compatibility with the reference implementation.
"""

import json

import eth_account
from hyperliquid.utils import signing as s

KEY = "0x0123456789012345678901234567890123456789012345678901234567890123"
WALLET = eth_account.Account.from_key(KEY)
VAULT = "0x1719884eb866cb12b2287399b15f7db5e7d775ea"
DEST = "0x5e9ee1089755c3435139848e47e6635505d5a13a"
NONCE = 1700000000000

ORDER = {
    "type": "order",
    "orders": [{"a": 4, "b": True, "p": "1670.1", "s": "0.0147", "r": False, "t": {"limit": {"tif": "Ioc"}}}],
    "grouping": "na",
}
L1 = [
    ("order", ORDER, None, None),
    ("order vault", ORDER, VAULT, None),
    ("order expiresAfter", ORDER, None, NONCE + 60000),
    ("order vault expiresAfter", ORDER, VAULT, NONCE + 60000),
    (
        "order trigger cloid builder",
        {
            "type": "order",
            "orders": [
                {
                    "a": 10107,
                    "b": False,
                    "p": "0.00012345",
                    "s": "100000",
                    "r": True,
                    "t": {"trigger": {"isMarket": True, "triggerPx": "0.0001", "tpsl": "sl"}},
                    "c": "0x00000000000000000000000000000001",
                }
            ],
            "grouping": "normalTpsl",
            "builder": {"b": DEST, "f": 10},
        },
        None,
        None,
    ),
    ("cancel", {"type": "cancel", "cancels": [{"a": 110000, "o": 123456789012}]}, None, None),
    ("updateIsolatedMargin negative", {"type": "updateIsolatedMargin", "asset": 1, "isBuy": True, "ntli": -1500000}, None, None),
    ("scheduleCancel", {"type": "scheduleCancel"}, None, None),
    ("dummy big int", {"type": "dummy", "num": 100000000000}, None, None),
    (
        "vaultModify nulls",
        {"type": "vaultModify", "vaultAddress": VAULT, "allowDeposits": None, "alwaysCloseOnWithdraw": False},
        None,
        None,
    ),
]

USER_SIGNED = [
    (
        "usdSend",
        "HyperliquidTransaction:UsdSend",
        s.USD_SEND_SIGN_TYPES,
        {"type": "usdSend", "destination": DEST, "amount": "1.5", "time": NONCE},
    ),
    (
        "withdraw3",
        "HyperliquidTransaction:Withdraw",
        s.WITHDRAW_SIGN_TYPES,
        {"type": "withdraw3", "destination": DEST, "amount": "10", "time": NONCE},
    ),
    (
        "usdClassTransfer",
        "HyperliquidTransaction:UsdClassTransfer",
        s.USD_CLASS_TRANSFER_SIGN_TYPES,
        {"type": "usdClassTransfer", "amount": "2 subaccount:" + VAULT, "toPerp": True, "nonce": NONCE},
    ),
    (
        "tokenDelegate",
        "HyperliquidTransaction:TokenDelegate",
        s.TOKEN_DELEGATE_TYPES,
        {"type": "tokenDelegate", "validator": DEST, "wei": 123000000, "isUndelegate": False, "nonce": NONCE},
    ),
]


def l1_vector(name, action, vault, expires_after, mainnet):
    h = s.action_hash(action, vault, NONCE, expires_after)
    return {
        "name": name,
        "action": action,
        "nonce": NONCE,
        "vault": vault,
        "expiresAfter": expires_after,
        "mainnet": mainnet,
        "connectionId": "0x" + h.hex(),
        "signature": s.sign_l1_action(WALLET, action, vault, NONCE, expires_after, mainnet),
    }


def user_signed_vector(name, primary_type, types, action, mainnet):
    signed = dict(action)
    signature = s.sign_user_signed_action(WALLET, signed, types, primary_type, mainnet)
    # Canonical wire order: type, signatureChainId, hyperliquidChain, fields..., nonce.
    wire = {"type": action["type"], "signatureChainId": signed["signatureChainId"], "hyperliquidChain": signed["hyperliquidChain"]}
    wire.update({k: v for k, v in action.items() if k != "type"})
    return {"name": name, "primaryType": primary_type, "types": types, "action": wire, "mainnet": mainnet, "signature": signature}


def multi_sig_vectors():
    multi_sig_user = VAULT
    outer = WALLET.address.lower()
    inner_l1 = s.sign_multi_sig_l1_action_payload(WALLET, ORDER, False, None, NONCE, None, multi_sig_user, outer)
    usd_send = {
        "type": "usdSend",
        "signatureChainId": "0x66eee",
        "hyperliquidChain": "Testnet",
        "destination": DEST,
        "amount": "1",
        "time": NONCE,
    }
    inner_us = s.sign_multi_sig_user_signed_action_payload(
        WALLET, usd_send, False, s.USD_SEND_SIGN_TYPES, "HyperliquidTransaction:UsdSend", multi_sig_user, outer
    )
    out = []
    for name, inner_action, inner_sig in [("l1 order", ORDER, inner_l1), ("usdSend", usd_send, inner_us)]:
        wrapper = {
            "type": "multiSig",
            "signatureChainId": "0x66eee",
            "signatures": [inner_sig],
            "payload": {"multiSigUser": multi_sig_user, "outerSigner": outer, "action": inner_action},
        }
        out.append(
            {
                "name": name,
                "multiSigUser": multi_sig_user,
                "outerSigner": outer,
                "nonce": NONCE,
                "mainnet": False,
                "innerSignature": inner_sig,
                "action": wrapper,
                "signature": s.sign_multi_sig_action(WALLET, wrapper, False, None, NONCE, None),
            }
        )
    return out


print(
    json.dumps(
        {
            "privateKey": KEY,
            "address": WALLET.address.lower(),
            "l1": [l1_vector(n, a, v, e, m) for (n, a, v, e) in L1 for m in (True, False)],
            "userSigned": [user_signed_vector(n, p, t, a, m) for (n, p, t, a) in USER_SIGNED for m in (True, False)],
            "multiSig": multi_sig_vectors(),
        },
        indent=1,
    )
)
