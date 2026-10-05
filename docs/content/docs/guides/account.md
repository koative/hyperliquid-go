---
title: Account state
weight: 2
---

Account state is public on Hyperliquid: every query takes the address of the
account and needs no key.

```go
info := hyperliquid.NewInfoClient(hyperliquid.Mainnet)
user := hyperliquid.MustParseAddress(os.Getenv("HL_ACCOUNT")) // the account, not an API wallet
```

{{< callout type="warning" >}}
Pass the address of the account (or sub-account, or vault) that holds the
funds. An API wallet only signs; querying its address returns an empty
account. See [API wallets](/docs/concepts/signing#api-wallets).
{{< /callout >}}

Perps on builder dexes are separate accounts: the perp queries below take a
`Dex` field, empty for the main dex.

## Positions and margin

[`ClearinghouseState`](/docs/reference#InfoClient.ClearinghouseState)
returns the perp account on one dex: account value, margin used, withdrawable
USDC and open positions. `Position.Szi` is the signed size, positive for long
and negative for short:

```go
state, err := info.ClearinghouseState(ctx, hyperliquid.ClearinghouseStateRequest{User: user})
if err != nil {
	log.Fatal(err)
}
fmt.Println("account value:", state.MarginSummary.AccountValue, "withdrawable:", state.Withdrawable)
for _, ap := range state.AssetPositions {
	p := ap.Position
	liq := "none"
	if p.LiquidationPx != nil {
		liq = p.LiquidationPx.String()
	}
	fmt.Printf("%-6s size=%s entry=%s upnl=%s lev=%d%s liq=%s\n",
		p.Coin, p.Szi, p.EntryPx, p.UnrealizedPnl, p.Leverage.Value, p.Leverage.Type, liq)
}
```

## Spot balances

[`SpotClearinghouseState`](/docs/reference#InfoClient.SpotClearinghouseState)
returns the spot balances. `Coin` is the token name, `Total` the balance and
`Hold` the part reserved by open orders:

```go
spot, err := info.SpotClearinghouseState(ctx, hyperliquid.SpotClearinghouseStateRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, b := range spot.Balances {
	fmt.Printf("%-8s total=%s hold=%s\n", b.Coin, b.Total, b.Hold)
}
```

## Open orders

[`OpenOrders`](/docs/reference#InfoClient.OpenOrders)
returns the resting orders of a dex.
[`FrontendOpenOrders`](/docs/reference#InfoClient.FrontendOpenOrders)
returns the same orders with the details the web app shows: order type,
trigger price and condition, reduce-only flag, time in force and attached
TP/SL children.

```go
orders, err := info.FrontendOpenOrders(ctx, hyperliquid.FrontendOpenOrdersRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, o := range orders {
	fmt.Printf("%d %s %s %s @ %s (%s, reduceOnly=%t)\n",
		o.Oid, o.Coin, o.Side, o.Sz, o.LimitPx, o.OrderType, o.ReduceOnly)
	if o.IsTrigger {
		fmt.Println("  triggers", o.TriggerCondition)
	}
}
```

`Side` is `"B"` for a buy and `"A"` for a sell. `Sz` is the remaining size and
`OrigSz` the size when placed.

## Order status

[`OrderStatus`](/docs/reference#InfoClient.OrderStatus)
looks up one order, open or not, by order ID or by
[client order ID](/docs/guides/orders#client-order-ids). It returns `nil`
if the exchange does not know the order:

```go
byOid, err := info.OrderStatus(ctx, hyperliquid.OrderStatusRequest{
	User: user,
	Oid:  hyperliquid.OrderRef{Oid: oid},
})
if err != nil {
	log.Fatal(err)
}

byCloid, err := info.OrderStatus(ctx, hyperliquid.OrderStatusRequest{
	User: user,
	Oid:  hyperliquid.OrderRef{Cloid: &cloid},
})
if err != nil {
	log.Fatal(err)
}

for _, s := range []*hyperliquid.OrderWithStatus{byOid, byCloid} {
	if s == nil {
		fmt.Println("unknown order")
		continue
	}
	fmt.Println(s.Order.Oid, s.Status, time.UnixMilli(s.StatusTimestamp).UTC())
}
```

`Status` is `"open"`, `"filled"`, `"canceled"`, `"triggered"`, `"rejected"` or a
more specific reason such as `"marginCanceled"` or `"reduceOnlyCanceled"`.
[`HistoricalOrders`](/docs/reference#InfoClient.HistoricalOrders)
returns the latest 2000 orders with their status.

## Fills

[`UserFills`](/docs/reference#InfoClient.UserFills) returns
up to 2000 of the most recent fills, newest first.
[`UserFillsByTime`](/docs/reference#InfoClient.UserFillsByTime)
returns up to 2000 fills in a time range, oldest first; page forward from the
last fill:

```go
start := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
var fills []hyperliquid.UserFill
for {
	page, err := info.UserFillsByTime(ctx, hyperliquid.UserFillsByTimeRequest{User: user, StartTime: start})
	if err != nil {
		log.Fatal(err)
	}
	fills = append(fills, page...)
	if len(page) < 2000 {
		break
	}
	start = page[len(page)-1].Time + 1
}
for _, f := range fills {
	fmt.Println(f.Coin, f.Dir, f.Sz, "@", f.Px, "fee", f.Fee, f.FeeToken, "pnl", f.ClosedPnl)
}
```

`Dir` describes the effect on the position (`"Open Long"`, `"Close Short"`,
...), `Crossed` reports whether the fill took liquidity, and a negative `Fee`
is a maker rebate. Set `AggregateByTime` to merge the partial fills of an
order that matched at the same time.

## Funding payments

[`UserFunding`](/docs/reference#InfoClient.UserFunding)
returns the funding payments of the account's positions. `Delta.USDC` is
negative when the account paid:

```go
payments, err := info.UserFunding(ctx, hyperliquid.UserFundingRequest{
	User:      user,
	StartTime: time.Now().Add(-24 * time.Hour).UnixMilli(),
})
if err != nil {
	log.Fatal(err)
}
for _, p := range payments {
	fmt.Println(p.Delta.Coin, p.Delta.USDC, "rate", p.Delta.FundingRate)
}
```

Deposits, withdrawals, transfers and liquidations are in
[`UserNonFundingLedgerUpdates`](/docs/reference#InfoClient.UserNonFundingLedgerUpdates).

## Portfolio history

[`Portfolio`](/docs/reference#InfoClient.Portfolio) returns
the account value and PnL history the web app charts, keyed by period:
`"day"`, `"week"`, `"month"`, `"allTime"`, and the perp-only `"perpDay"`,
`"perpWeek"`, `"perpMonth"` and `"perpAllTime"`:

```go
portfolio, err := info.Portfolio(ctx, hyperliquid.PortfolioRequest{User: user})
if err != nil {
	log.Fatal(err)
}
month := portfolio["month"]
if n := len(month.PnlHistory); n > 0 {
	fmt.Println("30-day PnL:", month.PnlHistory[n-1].Value, "volume:", month.Vlm)
}
```

## Fees and rate limits

[`UserFees`](/docs/reference#InfoClient.UserFees) returns
the account's current rates, the fee schedule and its daily volume. Rates are
fractions: `0.00045` is 4.5 basis points.
[`UserRateLimit`](/docs/reference#InfoClient.UserRateLimit)
returns the address-based action limit, which grows with traded volume:

```go
fees, err := info.UserFees(ctx, hyperliquid.UserFeesRequest{User: user})
if err != nil {
	log.Fatal(err)
}
fmt.Println("perp taker:", fees.UserCrossRate, "maker:", fees.UserAddRate)

limit, err := info.UserRateLimit(ctx, hyperliquid.UserRateLimitRequest{User: user})
if err != nil {
	log.Fatal(err)
}
fmt.Printf("actions used %d of %d\n", limit.NRequestsUsed, limit.NRequestsCap)
```

## Sub-accounts

[`SubAccounts`](/docs/reference#InfoClient.SubAccounts)
returns the sub-accounts of a master account with their main-dex perp and
spot state;
[`SubAccounts2`](/docs/reference#InfoClient.SubAccounts2)
includes every perp dex:

```go
subs, err := info.SubAccounts(ctx, hyperliquid.SubAccountsRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, s := range subs {
	fmt.Println(s.Name, s.SubAccountUser, s.ClearinghouseState.MarginSummary.AccountValue)
}
```

Trade for a sub-account with
[`WithVault`](/docs/concepts/nonces#vaults-and-sub-accounts).
