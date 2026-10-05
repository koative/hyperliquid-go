"""Generate signing golden vectors with the official Hyperliquid Python SDK.

Usage (from the repository root):

    pip install hyperliquid-python-sdk
    python testdata/gen_signing_vectors.py > testdata/signing_vectors.json

Every vector is produced by the SDK's own signing functions, so the Go tests
prove byte-for-byte compatibility with the reference implementation.
"""

import json

import eth_account
from hyperliquid.exchange import _multi_sig_payload_action
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


def sdk(fn):
    """Signs with the SDK's own signing function for the action."""
    return lambda action, mainnet: fn(WALLET, action, mainnet)


def nktkas(primary_type, types):
    """Signs an action the Python SDK lacks with types copied from nktkas/hyperliquid."""
    return lambda action, mainnet: s.sign_user_signed_action(WALLET, action, types, primary_type, mainnet)


def types(*fields):
    return [{"name": "hyperliquidChain", "type": "string"}] + [{"name": n, "type": t} for n, t in fields] + [
        {"name": "nonce", "type": "uint64"}
    ]


USDC = "USDC:0x6d1e7cde53ba9467b783cb7c530ce054"


def convert_signers(authorized_users, threshold):
    # Verbatim from Exchange.convert_to_multi_sig_user.
    return json.dumps({"authorizedUsers": sorted(authorized_users), "threshold": threshold})


SEND_TO_EVM_WITH_DATA = nktkas(
    "HyperliquidTransaction:SendToEvmWithData",
    types(
        ("token", "string"),
        ("amount", "string"),
        ("sourceDex", "string"),
        ("destinationRecipient", "string"),
        ("addressEncoding", "string"),
        ("destinationChainId", "uint32"),
        ("gasLimit", "uint64"),
        ("data", "bytes"),
    ),
)

# (name, signer, action in wire order without signatureChainId/hyperliquidChain)
USER_SIGNED = [
    ("usdSend", sdk(s.sign_usd_transfer_action), {"type": "usdSend", "destination": DEST, "amount": "1.5", "time": NONCE}),
    (
        "spotSend",
        sdk(s.sign_spot_transfer_action),
        {"type": "spotSend", "destination": DEST, "token": "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2", "amount": "0.5", "time": NONCE},
    ),
    ("withdraw3", sdk(s.sign_withdraw_from_bridge_action), {"type": "withdraw3", "destination": DEST, "amount": "10", "time": NONCE}),
    (
        "usdClassTransfer",
        sdk(s.sign_usd_class_transfer_action),
        {"type": "usdClassTransfer", "amount": "2", "toPerp": False, "nonce": NONCE},
    ),
    (
        "usdClassTransfer subaccount",
        sdk(s.sign_usd_class_transfer_action),
        {"type": "usdClassTransfer", "amount": "2 subaccount:" + VAULT, "toPerp": True, "nonce": NONCE},
    ),
    (
        "sendAsset",
        sdk(s.sign_send_asset_action),
        {
            "type": "sendAsset",
            "destination": DEST,
            "sourceDex": "",
            "destinationDex": "spot",
            "token": USDC,
            "amount": "3.25",
            "fromSubAccount": "",
            "nonce": NONCE,
        },
    ),
    (
        "sendAsset subaccount",
        sdk(s.sign_send_asset_action),
        {
            "type": "sendAsset",
            "destination": DEST,
            "sourceDex": "spot",
            "destinationDex": "xyz",
            "token": USDC,
            "amount": "3.25",
            "fromSubAccount": VAULT,
            "nonce": NONCE,
        },
    ),
    (
        "tokenDelegate",
        sdk(s.sign_token_delegate_action),
        {"type": "tokenDelegate", "validator": DEST, "wei": 123000000, "isUndelegate": False, "nonce": NONCE},
    ),
    ("approveAgent", sdk(s.sign_agent), {"type": "approveAgent", "agentAddress": DEST, "agentName": "bot", "nonce": NONCE}),
    ("approveAgent unnamed", sdk(s.sign_agent), {"type": "approveAgent", "agentAddress": DEST, "agentName": "", "nonce": NONCE}),
    (
        "approveBuilderFee",
        sdk(s.sign_approve_builder_fee),
        {"type": "approveBuilderFee", "maxFeeRate": "0.001%", "builder": DEST, "nonce": NONCE},
    ),
    (
        "convertToMultiSigUser",
        sdk(s.sign_convert_to_multi_sig_user_action),
        {"type": "convertToMultiSigUser", "signers": convert_signers([VAULT, DEST], 2), "nonce": NONCE},
    ),
    (
        "convertToMultiSigUser null",
        sdk(s.sign_convert_to_multi_sig_user_action),
        {"type": "convertToMultiSigUser", "signers": "null", "nonce": NONCE},
    ),
    (
        "userDexAbstraction",
        sdk(s.sign_user_dex_abstraction_action),
        {"type": "userDexAbstraction", "user": DEST, "enabled": True, "nonce": NONCE},
    ),
    (
        "userSetAbstraction",
        sdk(s.sign_user_set_abstraction_action),
        {"type": "userSetAbstraction", "user": DEST, "abstraction": "unifiedAccount", "nonce": NONCE},
    ),
    (
        "cDeposit",
        nktkas("HyperliquidTransaction:CDeposit", types(("wei", "uint64"))),
        {"type": "cDeposit", "wei": 100000000, "nonce": NONCE},
    ),
    (
        "cWithdraw",
        nktkas("HyperliquidTransaction:CWithdraw", types(("wei", "uint64"))),
        {"type": "cWithdraw", "wei": 100000000, "nonce": NONCE},
    ),
    (
        "linkStakingUser",
        nktkas("HyperliquidTransaction:LinkStakingUser", types(("user", "address"), ("isFinalize", "bool"))),
        {"type": "linkStakingUser", "user": DEST, "isFinalize": True, "nonce": NONCE},
    ),
    (
        "stakingLinkDisableTradingUser",
        nktkas("HyperliquidTransaction:StakingLinkDisableTradingUser", types(("tradingUser", "address"))),
        {"type": "stakingLinkDisableTradingUser", "tradingUser": DEST, "nonce": NONCE},
    ),
    (
        "userPortfolioMargin",
        nktkas("HyperliquidTransaction:UserPortfolioMargin", types(("user", "address"), ("enabled", "bool"))),
        {"type": "userPortfolioMargin", "user": DEST, "enabled": False, "nonce": NONCE},
    ),
    (
        "sendToEvmWithData",
        SEND_TO_EVM_WITH_DATA,
        {
            "type": "sendToEvmWithData",
            "token": "USDC",
            "amount": "1",
            "sourceDex": "",
            "destinationRecipient": DEST,
            "addressEncoding": "hex",
            "destinationChainId": 42161,
            "gasLimit": 200000,
            "data": "0xdeadbeef",
            "nonce": NONCE,
        },
    ),
    (
        "sendToEvmWithData empty data",
        SEND_TO_EVM_WITH_DATA,
        {
            "type": "sendToEvmWithData",
            "token": "USDC",
            "amount": "1",
            "sourceDex": "",
            "destinationRecipient": DEST,
            "addressEncoding": "hex",
            "destinationChainId": 42161,
            "gasLimit": 200000,
            "data": "0x",
            "nonce": NONCE,
        },
    ),
]

# Records the primary type and field list each signing function uses.
_captured = {}
_user_signed_payload = s.user_signed_payload


def _capture(primary_type, payload_types, action):
    _captured.update(primaryType=primary_type, types=payload_types)
    return _user_signed_payload(primary_type, payload_types, action)


s.user_signed_payload = _capture


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


def user_signed_vector(name, sign, action, mainnet):
    signed = dict(action)
    signature = sign(signed, mainnet)
    # Canonical wire order: type, signatureChainId, hyperliquidChain, fields..., nonce.
    wire = {"type": action["type"], "signatureChainId": signed["signatureChainId"], "hyperliquidChain": signed["hyperliquidChain"]}
    wire.update({k: v for k, v in action.items() if k != "type"})
    return {
        "name": name,
        "primaryType": _captured["primaryType"],
        "types": _captured["types"],
        "action": wire,
        "mainnet": mainnet,
        "signature": signature,
    }


def testnet_user_signed(action):
    """Returns action in wire order with the Testnet signing fields."""
    wire = {"type": action["type"], "signatureChainId": "0x66eee", "hyperliquidChain": "Testnet"}
    wire.update({k: v for k, v in action.items() if k != "type"})
    return wire


def multi_sig_vectors():
    multi_sig_user = VAULT
    outer = WALLET.address.lower()
    inners = [("l1 order", ORDER, s.sign_multi_sig_l1_action_payload(WALLET, ORDER, False, None, NONCE, None, multi_sig_user, outer))]
    for name, sign_types, primary_type, action in [
        (
            "usdSend",
            s.USD_SEND_SIGN_TYPES,
            "HyperliquidTransaction:UsdSend",
            {"type": "usdSend", "destination": DEST, "amount": "1", "time": NONCE},
        ),
        (
            "userSetAbstraction",
            s.USER_SET_ABSTRACTION_SIGN_TYPES,
            "HyperliquidTransaction:UserSetAbstraction",
            {"type": "userSetAbstraction", "user": multi_sig_user, "abstraction": "disabled", "nonce": NONCE},
        ),
        (
            "convertToMultiSigUser null",
            s.CONVERT_TO_MULTI_SIG_USER_SIGN_TYPES,
            "HyperliquidTransaction:ConvertToMultiSigUser",
            {"type": "convertToMultiSigUser", "signers": "null", "nonce": NONCE},
        ),
    ]:
        action = testnet_user_signed(action)
        sig = s.sign_multi_sig_user_signed_action_payload(WALLET, action, False, sign_types, primary_type, multi_sig_user, outer)
        inners.append((name, action, sig))
    out = []
    for name, inner_action, inner_sig in inners:
        wrapper = {
            "type": "multiSig",
            "signatureChainId": "0x66eee",
            "signatures": [inner_sig],
            # Exchange.multi_sig abbreviates userSetAbstraction's mode in the payload.
            "payload": {"multiSigUser": multi_sig_user, "outerSigner": outer, "action": _multi_sig_payload_action(inner_action)},
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
            "userSigned": [user_signed_vector(n, sign, a, m) for (n, sign, a) in USER_SIGNED for m in (True, False)],
            "multiSig": multi_sig_vectors(),
        },
        indent=1,
    )
)
