---
title: Markets and numbers
weight: 2
---

Exchange actions address markets by a numeric asset ID and take prices and
sizes as decimal strings that must fit the market's tick and lot size. Info
endpoints and subscriptions, on the other hand, take a coin name. This page
covers how to go from one to the other.

## Asset IDs

| Market | Asset ID | Coin name (info, WebSocket) |
|---|---|---|
| Perp on the main dex | index in `Meta.Universe` (BTC is `0`) | `"BTC"` |
| Spot pair | `10000` + pair index (`PURR/USDC` is `10000`) | `"PURR/USDC"` or `"@107"` |
| Perp on a builder dex (HIP-3) | `100000 + 10000*dexIndex` + index in that dex's universe | `"xyz:TSLA"` |
| Outcome side (HIP-4) | `100000000 + 10*outcome + side` | |

IDs follow list positions in the exchange metadata, so resolve them at
startup rather than hard-coding them.

## Resolving names with Markets

[`LoadMarkets`](/docs/reference#LoadMarkets) fetches the
main-dex, builder-dex and spot metadata in three requests and indexes every
market. [`Markets.Asset`](/docs/reference#Markets.Asset)
looks one up by name:

```go
markets, err := hyperliquid.LoadMarkets(ctx, info)
if err != nil {
	log.Fatal(err)
}
for _, name := range []string{"BTC", "PURR/USDC", "HYPE/USDC", "@107", "xyz:TSLA"} {
	a, ok := markets.Asset(name)
	if !ok {
		fmt.Println(name, "not listed")
		continue
	}
	fmt.Printf("%-10s id=%-6d coin=%-9s szDecimals=%d spot=%t dex=%q\n",
		name, a.ID, a.Name, a.SzDecimals, a.Spot, a.Dex)
}
```

On Mainnet this prints:

```text
BTC        id=0      coin=BTC       szDecimals=5 spot=false dex=""
PURR/USDC  id=10000  coin=PURR/USDC szDecimals=0 spot=true dex=""
HYPE/USDC  id=10107  coin=@107      szDecimals=2 spot=true dex=""
@107       id=10107  coin=@107      szDecimals=2 spot=true dex=""
xyz:TSLA   id=110001 coin=xyz:TSLA  szDecimals=3 spot=false dex="xyz"
```

An [`Asset`](/docs/reference#Asset) carries:

| Field | Meaning |
|---|---|
| `ID` | The asset ID to put in `Order.Asset`, `Cancel.Asset`, `UpdateLeverageAction.Asset` and so on. |
| `Name` | The coin name that info endpoints and subscriptions accept. |
| `SzDecimals` | Size precision; for spot, the base token's. Drives tick and lot rounding. |
| `Spot` | Whether the market is a spot pair. |
| `Dex` | The builder dex name for HIP-3 perps; empty for main-dex perps and spot. |

`Markets` is a snapshot and safe for concurrent reads. New listings do not
appear in it, so reload it periodically or when a lookup fails if your program
trades newly listed markets.

## Spot names

Spot pairs have two kinds of names on Hyperliquid. A few canonical pairs are
named `BASE/QUOTE` (`"PURR/USDC"`); every other pair is named `@<index>`
(`"@107"` is HYPE/USDC). That pair name, in
[`SpotPairMeta.Name`](/docs/reference#SpotPairMeta), is what
`AllMids`, `L2Book`, `CandleSnapshot`, fills and subscriptions use.

`Markets` also accepts `BASE/QUOTE` built from the token names, so
`markets.Asset("HYPE/USDC")` works, but the returned `Name` is still `"@107"`.
Always pass `Asset.Name`, not the name you looked up, to info requests:

```go
hype, ok := markets.Asset("HYPE/USDC")
if !ok {
	log.Fatal("HYPE/USDC not listed")
}
book, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: hype.Name}) // "@107"
if err != nil {
	log.Fatal(err)
}
fmt.Println("HYPE/USDC best bid:", book.Levels[0][0].Px)
```

Token names, such as `"HYPE"` in a
[`SpotBalance`](/docs/reference#SpotBalance), are not market
names. Transfers identify tokens as `"NAME:0x<tokenId>"`, built from
[`SpotTokenMeta`](/docs/reference#SpotTokenMeta).

## Builder dexes

Builder-deployed perp dexes (HIP-3) prefix their coins with the dex name, as
in `"xyz:TSLA"`.
[`InfoClient.PerpDexs`](/docs/reference#InfoClient.PerpDexs)
lists them; element 0 is `nil` and stands for the main dex, and a dex's index
in that list is the `dexIndex` of its asset IDs. Requests that cover a whole
dex take a `Dex` field, empty for the main dex:

```go
mids, err := info.AllMids(ctx, hyperliquid.AllMidsRequest{Dex: "xyz"})
if err != nil {
	log.Fatal(err)
}
fmt.Println("xyz:TSLA mid:", mids["xyz:TSLA"])
```

Orders on a builder dex are ordinary orders with the builder-dex asset ID.
Each builder dex margins positions from its own balance: move collateral to
it with [`SendAsset`](/docs/reference#ExchangeClient.SendAsset),
or let it use the main dex's collateral with dex abstraction (see
[`InfoClient.UserDexAbstraction`](/docs/reference#InfoClient.UserDexAbstraction)).

## Decimal

Prices, sizes and amounts are
[`Decimal`](/docs/reference#Decimal) values: strings in
Hyperliquid's wire form, such as `"0.001"` or `"-12.5"`. Nothing is parsed into
a float on the way in, so no precision is lost, and string constants can be
used directly:

```go
order := hyperliquid.Order{Asset: eth.ID, IsBuy: true, Price: "2500.5", Size: "0.01"}

mids, err := info.AllMids(ctx, hyperliquid.AllMidsRequest{})
if err != nil {
	log.Fatal(err)
}
mid, err := mids["ETH"].Float64() // for arithmetic
if err != nil {
	log.Fatal(err)
}
px := hyperliquid.DecimalFromFloat(mid * 0.99) // back to a Decimal; round it before use

size, err := hyperliquid.ParseDecimal(os.Getenv("ORDER_SIZE")) // validate untrusted input
if err != nil {
	log.Fatal(err)
}
```

Use `Float64` or a decimal library of your choice for arithmetic.
[`IsZero`](/docs/reference#Decimal.IsZero) reports whether a
value is empty or numerically zero. When an action is encoded, each `Decimal`
is normalized (`"1.50"` becomes `"1.5"`) and the signature covers the
normalized form, which is what the exchange expects. An invalid `Decimal`
fails the request before it is signed.

## Tick and lot sizes

The exchange rejects prices and sizes that are too precise
([Hyperliquid docs](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size)):

- **Size** has at most `szDecimals` decimal places.
- **Price** has at most 5 significant figures, and at most `6 - szDecimals`
  decimal places for perps or `8 - szDecimals` for spot. Integer prices are
  always valid, whatever their number of digits.

[`FormatPrice`](/docs/reference#FormatPrice) and
[`FormatSize`](/docs/reference#FormatSize) apply these rules,
and [`Asset.FormatPrice`](/docs/reference#Asset.FormatPrice)
and [`Asset.FormatSize`](/docs/reference#Asset.FormatSize)
fill in the asset's decimals:

```go
px, err := eth.FormatPrice("2712.3456") // ETH perp: szDecimals 4
if err != nil {
	log.Fatal(err)
}
sz, err := eth.FormatSize("0.123456")
if err != nil {
	log.Fatal(err)
}
fmt.Println(px, sz) // 2712.3 0.1234
```

| Call | Result |
|---|---|
| `FormatPrice("2712.3456", 4, false)` | `2712.3` (5 significant figures) |
| `FormatPrice("123456.7", 5, false)` | `123456` (integer digits are kept) |
| `FormatPrice("0.0001234567", 0, false)` | `0.000123` (6 decimals for a perp) |
| `FormatPrice("0.00001234567", 0, true)` | `0.00001234` (8 decimals for spot) |
| `FormatSize("0.00004", 4)` | error `ErrTruncatedToZero` |

Both functions round **toward zero**, never up: a size never exceeds what you
asked for, and a price moves toward zero whichever side you are on. Round in
your own code first if you need another direction. When nothing significant
is left after rounding
they return
[`ErrTruncatedToZero`](/docs/reference#ErrTruncatedToZero)
rather than a zero that the exchange would reject.
