---
title: Transfers
weight: 5
---

Moving funds between addresses, between the perp and spot balances, across
perp dexes and off Hyperliquid. Every method here returns only an error;
check the result with `InfoClient.ClearinghouseState`,
`SpotClearinghouseState` or `UserNonFundingLedgerUpdates`.

## Which key signs

Transfers are *user-signed* actions: they must be signed by the account's own
key, not an API (agent) wallet. An agent wallet can trade but cannot move
funds; the exchange rejects a transfer it signs. See
[Signing and keys](/docs/concepts/signing).

```go
owner, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_ACCOUNT_KEY"))
if err != nil {
	log.Fatal(err)
}
ex := hyperliquid.NewExchangeClient(hyperliquid.Testnet, owner)
```

[`ExchangeClient.WithVault`](/docs/reference#ExchangeClient.WithVault)
does not apply to user-signed actions; they always act for the signer. To
move a sub-account's funds, use the sub-account fields shown below.

## Amounts

The transfers on this page take amounts as [`Decimal`](/docs/reference#Decimal)
strings in token units: `"25"` or `"0.5"`. Some other actions take integer
fields instead, documented on each field:

| Field | Unit | Used by |
|---|---|---|
| `Amount Decimal` | Token units, e.g. `"25.5"` USDC | `USDSend`, `SpotSend`, `Withdraw`, `USDClassTransfer`, `SendAsset`, `SubAccountSpotTransfer` |
| `USD uint64` | USDC × 10⁶ (`25_500_000` = 25.5 USDC) | `SubAccountTransfer`, `VaultTransfer`, `VaultDistribute`, `CreateVault` (`InitialUSD`) |
| `Wei uint64` | HYPE × 10⁸ | `CDeposit`, `CWithdraw`, `TokenDelegate` |

## Send USDC

[`USDSend`](/docs/reference#ExchangeClient.USDSend) sends
USDC from your perp balance to another address on Hyperliquid:

```go
err := ex.USDSend(ctx, hyperliquid.USDSendAction{
	Destination: hyperliquid.MustParseAddress("0x0000000000000000000000000000000000000001"),
	Amount:      "10",
})
if err != nil {
	log.Fatal(err)
}
```

## Send a spot token

[`SpotSend`](/docs/reference#ExchangeClient.SpotSend) names
the token as `"NAME:0x<tokenId>"`. Build it from
[`SpotMeta`](/docs/reference#InfoClient.SpotMeta):

```go
spot, err := info.SpotMeta(ctx)
if err != nil {
	log.Fatal(err)
}
var purr string
for _, t := range spot.Tokens {
	if t.Name == "PURR" {
		purr = t.Name + ":" + t.TokenID
	}
}

err = ex.SpotSend(ctx, hyperliquid.SpotSendAction{
	Destination: destination,
	Token:       purr,
	Amount:      "100",
})
if err != nil {
	log.Fatal(err)
}
```

## Withdraw to Arbitrum

[`Withdraw`](/docs/reference#ExchangeClient.Withdraw)
sends USDC from the perp balance through the bridge to an address on
Arbitrum. A fixed fee is deducted from the amount.

```go
err := ex.Withdraw(ctx, hyperliquid.WithdrawAction{
	Destination: owner.Address(),
	Amount:      "50",
})
if err != nil {
	log.Fatal(err)
}
```

## Move USDC between perp and spot

[`USDClassTransfer`](/docs/reference#ExchangeClient.USDClassTransfer)
moves USDC between your spot and perp balances. `ToPerp` picks the
direction:

```go
// Spot to perp.
err := ex.USDClassTransfer(ctx, hyperliquid.USDClassTransferAction{Amount: "20", ToPerp: true})
if err != nil {
	log.Fatal(err)
}

// Perp to spot, on one of your sub-accounts.
err = ex.USDClassTransfer(ctx, hyperliquid.USDClassTransferAction{
	Amount:     "20",
	ToPerp:     false,
	SubAccount: &subAccount,
})
if err != nil {
	log.Fatal(err)
}
```

## Move tokens across dexes, spot and accounts

[`SendAsset`](/docs/reference#ExchangeClient.SendAsset)
is the general transfer: any token, any source and destination balance, to
yourself or another address. `SourceDex` and `DestinationDex` name the
balance:

- `""`: the main perp dex
- `"spot"`: the spot balance
- any other name: a builder-deployed (HIP-3) perp dex, such as `"xyz"`

When a perp dex is involved, the token must be that dex's collateral token.

```go
usdc := "USDC:0xeb62eee3685fc4c43992febcd9e75443" // Testnet; see SpotMeta

// Fund the Testnet "test" dex (USDC-collateralized) from your main perp balance.
err := ex.SendAsset(ctx, hyperliquid.SendAssetAction{
	Destination:    owner.Address(),
	SourceDex:      "",
	DestinationDex: "test",
	Token:          usdc,
	Amount:         "100",
})
if err != nil {
	log.Fatal(err)
}

// Send spot USDC from a sub-account to another address.
err = ex.SendAsset(ctx, hyperliquid.SendAssetAction{
	Destination:    destination,
	SourceDex:      "spot",
	DestinationDex: "spot",
	Token:          usdc,
	Amount:         "5",
	FromSubAccount: &subAccount,
})
if err != nil {
	log.Fatal(err)
}
```

[`AgentSendAsset`](/docs/reference#ExchangeClient.AgentSendAsset)
takes the same fields but is an L1 action that an API wallet may sign.

Sub-account USDC and spot transfers have their own actions; see
[Vaults and sub-accounts](/docs/guides/vaults-and-subaccounts).

## Move spot tokens to the HyperEVM

A spot token linked to an ERC-20 contract moves to the HyperEVM when you
send it to the token's system address: `0x20` followed by zeros and the
token index in big-endian (HYPE uses `0x2222…2222`). The tokens arrive at
the same address on the EVM side.

```go
var system hyperliquid.Address
system[0] = 0x20
binary.BigEndian.PutUint64(system[12:], uint64(tokenIndex)) // SpotTokenMeta.Index

err := ex.SendAsset(ctx, hyperliquid.SendAssetAction{
	Destination:    system,
	SourceDex:      "spot",
	DestinationDex: "spot",
	Token:          token, // "NAME:0x<tokenId>"
	Amount:         "1",
})
if err != nil {
	log.Fatal(err)
}
```

See [HyperCore ↔ HyperEVM transfers](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/hyperevm/hypercore-less-than-greater-than-hyperevm-transfers)
for linking and the caveats.

## Send to a contract with data

[`SendToEVMWithData`](/docs/reference#ExchangeClient.SendToEVMWithData)
sends a token from HyperCore to a contract that implements
`ICoreReceiveWithData`, passing it a payload. `Token` is the plain token
name here, and `DestinationRecipient` is encoded per `AddressEncoding`:

```go
err := ex.SendToEVMWithData(ctx, hyperliquid.SendToEVMWithDataAction{
	Token:                "USDC",
	Amount:               "1",
	SourceDex:            "",
	DestinationRecipient: "0x5e9ee1089755c3435139848e47e6635505d5a13a",
	AddressEncoding:      hyperliquid.AddressEncodingHex,
	DestinationChainID:   42161,
	GasLimit:             200000,
	Data:                 []byte{0xde, 0xad, 0xbe, 0xef},
})
if err != nil {
	log.Fatal(err)
}
```
