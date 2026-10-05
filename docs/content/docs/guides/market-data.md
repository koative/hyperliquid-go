---
title: Market data
weight: 1
---

Market data comes from
[`InfoClient`](/docs/reference#InfoClient), which needs no
keys. Coins are named as in [Markets](/docs/concepts/markets):
`"BTC"` for perps, `"PURR/USDC"` or `"@107"` for spot pairs and `"xyz:TSLA"`
for builder-dex perps. Prices and sizes are
[`Decimal`](/docs/concepts/markets#decimal) strings and
timestamps are Unix milliseconds.

```go
info := hyperliquid.NewInfoClient(hyperliquid.Mainnet)
```

For live updates, subscribe to the same data over the
[WebSocket](/docs/guides/websocket) instead of polling.

## Mid prices

[`AllMids`](/docs/reference#InfoClient.AllMids) returns the
mid price of every perp and spot coin, keyed by coin name. Set `Dex` for a
builder dex:

```go
mids, err := info.AllMids(ctx, hyperliquid.AllMidsRequest{})
if err != nil {
	log.Fatal(err)
}
fmt.Println("BTC:", mids["BTC"], "HYPE/USDC:", mids["@107"])
```

## Perp metadata and asset contexts

[`Meta`](/docs/reference#InfoClient.Meta) lists the perps of
a dex: name, size decimals, maximum leverage and margin table. An asset's
position in `Universe` is its asset ID on the main dex.
[`MetaAndAssetCtxs`](/docs/reference#InfoClient.MetaAndAssetCtxs)
adds the live state of each asset (mark, oracle and mid price, funding, open
interest, volume) in `AssetCtxs`, parallel to `Universe`:

```go
m, err := info.MetaAndAssetCtxs(ctx, hyperliquid.MetaAndAssetCtxsRequest{})
if err != nil {
	log.Fatal(err)
}
for i, asset := range m.Meta.Universe {
	if asset.IsDelisted {
		continue
	}
	c := m.AssetCtxs[i]
	fmt.Printf("%-6s mark=%s funding=%s oi=%s maxLev=%d\n",
		asset.Name, c.MarkPx, c.Funding, c.OpenInterest, asset.MaxLeverage)
}
```

`MidPx`, `Premium` and `ImpactPxs` are pointers because they are `null` for
illiquid assets. Delisted assets stay in `Universe` so that the indexes, and
with them the asset IDs, do not shift.

## Order book

[`L2Book`](/docs/reference#InfoClient.L2Book) returns up to
20 levels per side, best first: bids in `Levels[0]`, asks in `Levels[1]`. It
returns `nil` for a coin that does not exist. Set `NSigFigs` (2 to 5) to
aggregate levels into coarser price buckets:

```go
book, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: "ETH"})
if err != nil {
	log.Fatal(err)
}
if book == nil {
	log.Fatal("unknown coin")
}
bids, asks := book.Levels[0], book.Levels[1]
fmt.Printf("bid %s x %s | ask %s x %s\n", bids[0].Px, bids[0].Sz, asks[0].Px, asks[0].Sz)

sigFigs := 3
coarse, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: "BTC", NSigFigs: &sigFigs})
if err != nil {
	log.Fatal(err)
}
fmt.Println("BTC bucketed best bid:", coarse.Levels[0][0].Px)
```

## Candles

[`CandleSnapshot`](/docs/reference#InfoClient.CandleSnapshot)
returns the candles whose open time falls in a range. Only the 5000 most
recent candles of each interval are available:

```go
candles, err := info.CandleSnapshot(ctx, hyperliquid.CandleSnapshotRequest{
	Coin:      "BTC",
	Interval:  hyperliquid.Candle1h,
	StartTime: time.Now().Add(-24 * time.Hour).UnixMilli(),
	// EndTime zero means now.
})
if err != nil {
	log.Fatal(err)
}
for _, c := range candles {
	fmt.Println(time.UnixMilli(c.OpenTime).UTC().Format(time.DateTime), c.Open, c.High, c.Low, c.Close, c.Volume)
}
```

## Recent trades

[`RecentTrades`](/docs/reference#InfoClient.RecentTrades)
returns the latest public trades of a coin. `Side` is the aggressor's side
and `Users` holds the buyer and the seller:

```go
trades, err := info.RecentTrades(ctx, hyperliquid.RecentTradesRequest{Coin: "BTC"})
if err != nil {
	log.Fatal(err)
}
for _, t := range trades {
	fmt.Println(t.Side, t.Sz, "@", t.Px, time.UnixMilli(t.Time).UTC().Format(time.TimeOnly))
}
```

## Funding

[`FundingHistory`](/docs/reference#InfoClient.FundingHistory)
returns historical funding rates of a perp, oldest first, at most 500 per
call. Page forward from the last timestamp:

```go
start := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
var rates []hyperliquid.FundingRate
for {
	page, err := info.FundingHistory(ctx, hyperliquid.FundingHistoryRequest{Coin: "ETH", StartTime: start})
	if err != nil {
		log.Fatal(err)
	}
	rates = append(rates, page...)
	if len(page) < 500 {
		break
	}
	start = page[len(page)-1].Time + 1
}
fmt.Println(len(rates), "hourly rates, latest", rates[len(rates)-1].FundingRate)
```

[`PredictedFundings`](/docs/reference#InfoClient.PredictedFundings)
returns the next funding rate of each coin on Hyperliquid and on other venues,
keyed by coin and then by venue (`"HlPerp"`, `"BinPerp"`, `"BybitPerp"`, ...).
A `nil` entry means the venue has no prediction for that coin:

```go
predicted, err := info.PredictedFundings(ctx)
if err != nil {
	log.Fatal(err)
}
for venue, p := range predicted["BTC"] {
	if p != nil {
		fmt.Println(venue, p.FundingRate, "next at", time.UnixMilli(p.NextFundingTime).UTC())
	}
}
```

The response's JSON arrays of `[key, value]` pairs decode into a
[`TupleMap`](/docs/reference#TupleMap), an ordinary Go map.

## Spot metadata

[`SpotMeta`](/docs/reference#InfoClient.SpotMeta) lists the
spot tokens and pairs. A pair names its base and quote tokens by token index;
its asset ID is `10000 + Index` and its coin name is `Name`:

```go
spot, err := info.SpotMeta(ctx)
if err != nil {
	log.Fatal(err)
}
tokens := make(map[int]hyperliquid.SpotTokenMeta, len(spot.Tokens))
for _, t := range spot.Tokens {
	tokens[t.Index] = t
}
for _, p := range spot.Universe[:5] {
	base, quote := tokens[p.Tokens[0]], tokens[p.Tokens[1]]
	fmt.Printf("%-10s %s/%s asset=%d\n", p.Name, base.Name, quote.Name, 10000+p.Index)
}
```

[`SpotMetaAndAssetCtxs`](/docs/reference#InfoClient.SpotMetaAndAssetCtxs)
adds prices, volume and supply per pair. To resolve names to asset IDs, use
[`LoadMarkets`](/docs/concepts/markets#resolving-names-with-markets)
rather than building your own index.

## Builder-deployed perp dexes

[`PerpDexs`](/docs/reference#InfoClient.PerpDexs) lists the
HIP-3 dexes. Element 0 is `nil` and stands for the main dex; the others
describe each dex, and their index is the `dexIndex` in their asset IDs.
Requests that cover a dex, such as `Meta`, `MetaAndAssetCtxs` and `AllMids`,
take its name in `Dex`.
[`AllPerpMetas`](/docs/reference#InfoClient.AllPerpMetas)
fetches the `Meta` of every dex in one call, indexed like `PerpDexs`:

```go
dexs, err := info.PerpDexs(ctx)
if err != nil {
	log.Fatal(err)
}
for i, d := range dexs {
	if d == nil {
		continue // the main dex
	}
	meta, err := info.Meta(ctx, hyperliquid.MetaRequest{Dex: d.Name})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d %-5s %-22s %d perps, asset IDs from %d\n",
		i, d.Name, d.FullName, len(meta.Universe), 100000+10000*i)
}
```

## Explorer

The explorer requests go to the network's `RPCURL` over HTTP, even when the
client sends its other requests [over the WebSocket](/docs/reference#WithWebSocket).
[`UserDetails`](/docs/reference#InfoClient.UserDetails)
returns a user's recent transactions,
[`TxDetails`](/docs/reference#InfoClient.TxDetails) one
transaction by hash, and
[`BlockDetails`](/docs/reference#InfoClient.BlockDetails)
an L1 block with its transactions:

```go
txs, err := info.UserDetails(ctx, hyperliquid.UserDetailsRequest{User: user})
if err != nil {
	log.Fatal(err)
}
if len(txs) == 0 {
	log.Fatal("no transactions")
}
for _, tx := range txs {
	if tx.Error != nil {
		fmt.Println(tx.Hash, "failed:", *tx.Error)
	}
}

tx, err := info.TxDetails(ctx, hyperliquid.TxDetailsRequest{Hash: txs[0].Hash})
if err != nil {
	log.Fatal(err)
}
fmt.Println(tx.Hash, "in block", tx.Block, string(tx.Action))

block, err := info.BlockDetails(ctx, hyperliquid.BlockDetailsRequest{Height: tx.Block})
if err != nil {
	log.Fatal(err)
}
fmt.Println("block", block.Height, "has", block.NumTxs, "txs, proposer", block.Proposer)
```

`ExplorerTx.Action` is the raw action JSON, such as `{"type":"order",...}`.
To follow new blocks and transactions as they happen, use
[`DialExplorerWebSocket`](/docs/reference#DialExplorerWebSocket).
