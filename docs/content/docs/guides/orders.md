---
title: Orders
weight: 3
---

Orders, cancels and the other trading actions are L1 actions: an
[API wallet](/docs/concepts/signing#api-wallets) can sign them, and they act
for the account that approved it. The snippets on this page build on this
setup and use Testnet:

```go
signer, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_AGENT_KEY"))
if err != nil {
	log.Fatal(err)
}
info := hyperliquid.NewInfoClient(hyperliquid.Testnet)
ex := hyperliquid.NewExchangeClient(hyperliquid.Testnet, signer)
user := hyperliquid.MustParseAddress(os.Getenv("HL_ACCOUNT")) // the account, for queries

markets, err := hyperliquid.LoadMarkets(ctx, info)
if err != nil {
	log.Fatal(err)
}
eth, ok := markets.Asset("ETH")
if !ok {
	log.Fatal("ETH not listed")
}
```

Orders take the [asset ID](/docs/concepts/markets#asset-ids), and prices and
sizes rounded to the market's
[tick and lot size](/docs/concepts/markets#tick-and-lot-sizes).

## Limit orders

An [`Order`](/docs/reference#Order) with a
[`LimitOrder`](/docs/reference#LimitOrder) type rests on the book according
to its time in force:

| `Tif` | Behavior |
|---|---|
| `TifGtc` | Rests until filled or canceled. |
| `TifAlo` | Add liquidity only: canceled instead of taking liquidity (post-only). |
| `TifIoc` | Fills what it can immediately and cancels the rest. |

```go
px, err := eth.FormatPrice("2500")
if err != nil {
	log.Fatal(err)
}
sz, err := eth.FormatSize("0.01")
if err != nil {
	log.Fatal(err)
}
results, err := ex.Order(ctx, hyperliquid.OrderAction{
	Orders: []hyperliquid.Order{{
		Asset: eth.ID,
		IsBuy: true,
		Price: px,
		Size:  sz,
		Type:  hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifAlo}},
	}},
})
```

One `OrderAction` can carry many orders; they are accepted or rejected one by
one. Set `ReduceOnly` on an order that must only shrink a position.

## Handling results

[`ExchangeClient.Order`](/docs/reference#ExchangeClient.Order) returns one
[`OrderResult`](/docs/reference#OrderResult) per order, in request order.
Exactly one of its fields is set:

```go
results, err := ex.Order(ctx, action)
if _, rejected := errors.AsType[*hyperliquid.StatusError](err); err != nil && !rejected {
	log.Fatal(err) // the whole action failed; see the errors page
}
for i, r := range results {
	switch {
	case r.Resting != nil:
		fmt.Printf("order %d resting, oid %d\n", i, r.Resting.Oid)
	case r.Filled != nil:
		fmt.Printf("order %d filled %s @ %s, oid %d\n", i, r.Filled.TotalSz, r.Filled.AvgPx, r.Filled.Oid)
	case r.Error != "":
		fmt.Printf("order %d rejected: %s\n", i, r.Error)
	default:
		fmt.Printf("order %d: %s\n", i, r.Status) // "waitingForFill", "waitingForTrigger"
	}
}
```

Rejected orders are also reported as a joined
[`*StatusError`](/docs/reference#StatusError) per order, so a non-nil error
does not mean that nothing was placed. A filled order gets an `oid` too, which
fills and order queries refer to. See [Errors](/docs/concepts/errors) for the
other error kinds and for deciding whether an action executed when the
response was lost.

## Market orders

Hyperliquid has no market order type. Send an IOC limit order with a price
far enough through the book to fill, which also caps your slippage:

```go
mids, err := info.AllMids(ctx, hyperliquid.AllMidsRequest{})
if err != nil {
	log.Fatal(err)
}
mid, err := mids[eth.Name].Float64()
if err != nil {
	log.Fatal(err)
}
const slippage = 0.01 // accept up to 1% worse than mid
px, err := eth.FormatPrice(hyperliquid.DecimalFromFloat(mid * (1 + slippage)))
if err != nil {
	log.Fatal(err)
}
results, err := ex.Order(ctx, hyperliquid.OrderAction{
	Orders: []hyperliquid.Order{{
		Asset: eth.ID,
		IsBuy: true,
		Price: px,
		Size:  "0.01",
		Type:  hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifIoc}},
	}},
})
```

For a sell, use `mid * (1 - slippage)`. An IOC order that finds no liquidity
within its price is rejected rather than left resting.

## Client order IDs

A [`Cloid`](/docs/reference#Cloid) is a 16-byte ID you choose. Set one on
every order to look it up, cancel it, and reconcile after a lost response
without knowing the exchange's `oid`:

```go
var id hyperliquid.Cloid
rand.Read(id[:]) // crypto/rand

order := hyperliquid.Order{
	Asset: eth.ID, IsBuy: true, Price: "2500", Size: "0.01",
	Type:  hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifGtc}},
	Cloid: &id,
}
if _, err := ex.Order(ctx, hyperliquid.OrderAction{Orders: []hyperliquid.Order{order}}); err != nil {
	log.Fatal(err)
}

status, err := info.OrderStatus(ctx, hyperliquid.OrderStatusRequest{
	User: user,
	Oid:  hyperliquid.OrderRef{Cloid: &id},
})
if err != nil {
	log.Fatal(err)
}
if status != nil {
	fmt.Println(id, "is", status.Status)
}
```

[`ParseCloid`](/docs/reference#ParseCloid) reads one from its `0x`-prefixed
hex form, and fills, open orders and order updates carry it back.

## Take profit and stop loss

A trigger order rests off the book until the mark price crosses `TriggerPx`.
`Tpsl` says whether it is a take profit or a stop loss, and `IsMarket`
executes it as a market order when triggered; otherwise `Price` is its limit
price. TP/SL orders are normally reduce-only and on the opposite side of the
position.

The `Grouping` of the action links them to a position:

- [`GroupingNormalTpsl`](/docs/reference#GroupingNormalTpsl) attaches TP/SL
  orders of a fixed size to the first order of the action, typically an
  entry order.
- [`GroupingPositionTpsl`](/docs/reference#GroupingPositionTpsl) attaches
  TP/SL orders that track the size of the existing position.

An entry with a bracket:

```go
size := hyperliquid.Decimal("0.01")
trigger := func(px hyperliquid.Decimal, kind hyperliquid.Tpsl) hyperliquid.Order {
	return hyperliquid.Order{
		Asset: eth.ID, IsBuy: false, Price: px, Size: size, ReduceOnly: true,
		Type: hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrder{
			IsMarket: true, TriggerPx: px, Tpsl: kind,
		}},
	}
}
results, err := ex.Order(ctx, hyperliquid.OrderAction{
	Orders: []hyperliquid.Order{
		{ // entry
			Asset: eth.ID, IsBuy: true, Price: "2500", Size: size,
			Type: hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifGtc}},
		},
		trigger("2750", hyperliquid.TpslTakeProfit),
		trigger("2400", hyperliquid.TpslStopLoss),
	},
	Grouping: hyperliquid.GroupingNormalTpsl,
})
```

A stop loss on a position you already hold, sized from the account state:

```go
state, err := info.ClearinghouseState(ctx, hyperliquid.ClearinghouseStateRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, ap := range state.AssetPositions {
	p := ap.Position
	if p.Coin != eth.Name {
		continue
	}
	long := !strings.HasPrefix(string(p.Szi), "-")
	stopPx, err := eth.FormatPrice("2400") // below the entry of a long
	if err != nil {
		log.Fatal(err)
	}
	_, err = ex.Order(ctx, hyperliquid.OrderAction{
		Orders: []hyperliquid.Order{{
			Asset: eth.ID, IsBuy: !long, Price: stopPx,
			Size:       hyperliquid.Decimal(strings.TrimPrefix(string(p.Szi), "-")),
			ReduceOnly: true,
			Type: hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrder{
				IsMarket: true, TriggerPx: stopPx, Tpsl: hyperliquid.TpslStopLoss,
			}},
		}},
		Grouping: hyperliquid.GroupingPositionTpsl,
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Modifying orders

[`Modify`](/docs/reference#ExchangeClient.Modify) replaces a resting order,
identified by `oid` or by cloid, with a new one. Unless `AlwaysPlace` is set,
the new order must be a non-trigger ALO order or a GTC order that would not
take liquidity, and it is not placed if canceling the old one fails:

```go
err := ex.Modify(ctx, hyperliquid.ModifyAction{
	Oid: hyperliquid.OrderRef{Oid: oid},
	Order: hyperliquid.Order{
		Asset: eth.ID, IsBuy: true, Price: "2490", Size: "0.01",
		Type: hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifAlo}},
	},
})
```

[`BatchModify`](/docs/reference#ExchangeClient.BatchModify) replaces several
orders in one action and returns one `OrderResult` per new order, like
`Order`:

```go
results, err := ex.BatchModify(ctx, hyperliquid.BatchModifyAction{
	Modifies: []hyperliquid.Modify{
		{Oid: hyperliquid.OrderRef{Oid: oid}, Order: order},
		{Oid: hyperliquid.OrderRef{Cloid: &cloid}, Order: order},
	},
})
```

## Canceling orders

[`Cancel`](/docs/reference#ExchangeClient.Cancel) cancels by asset and `oid`,
[`CancelByCloid`](/docs/reference#ExchangeClient.CancelByCloid) by asset and
client order ID. A cancel that fails, for example because the order already
filled, is reported as a joined `*StatusError` per item:

```go
err := ex.Cancel(ctx, hyperliquid.CancelAction{
	Cancels: []hyperliquid.Cancel{{Asset: eth.ID, Oid: oid}},
})
if err != nil {
	log.Print(err)
}

err = ex.CancelByCloid(ctx, hyperliquid.CancelByCloidAction{
	Cancels: []hyperliquid.CancelByCloid{{Asset: eth.ID, Cloid: cloid}},
})
if err != nil {
	log.Print(err)
}
```

To cancel everything on a market, list the open orders with
[`InfoClient.OpenOrders`](/docs/reference#InfoClient.OpenOrders) and cancel
them in one action.

## Dead man's switch

[`ScheduleCancel`](/docs/reference#ExchangeClient.ScheduleCancel) cancels all
open orders at a given time, at least 5 seconds ahead. Push the time forward
while your process is healthy; if it dies or loses connectivity, its orders
are canceled. A zero `Time` clears the schedule. The switch may fire at most
10 times per day.

```go
ticker := time.NewTicker(10 * time.Second)
defer ticker.Stop()
for {
	at := time.Now().Add(30 * time.Second).UnixMilli()
	if err := ex.ScheduleCancel(ctx, hyperliquid.ScheduleCancelAction{Time: at}); err != nil {
		log.Print(err)
	}
	select {
	case <-ticker.C:
	case <-ctx.Done():
		return
	}
}
```

Each refresh is an action and counts against your rate limit.

## TWAP orders

[`TwapOrder`](/docs/reference#ExchangeClient.TwapOrder) trades a total size
over 5 to 1440 minutes in slices and returns the TWAP ID;
[`TwapCancel`](/docs/reference#ExchangeClient.TwapCancel) stops it. Both
return a bare `*StatusError` when the exchange rejects them:

```go
twapID, err := ex.TwapOrder(ctx, hyperliquid.TwapOrderAction{
	Twap: hyperliquid.Twap{
		Asset:     eth.ID,
		IsBuy:     true,
		Size:      "1",
		Minutes:   60,
		Randomize: true,
	},
})
if err != nil {
	log.Fatal(err)
}

err = ex.TwapCancel(ctx, hyperliquid.TwapCancelAction{Asset: eth.ID, TwapID: twapID})
if err != nil {
	log.Fatal(err)
}
```

`Details` optionally delays the start until the mark price crosses a price
and stops the TWAP at a stop price. Follow progress with
[`InfoClient.TwapHistory`](/docs/reference#InfoClient.TwapHistory) and
[`UserTwapSliceFills`](/docs/reference#InfoClient.UserTwapSliceFills).

## Trailing stops

[`TrailingStop`](/docs/reference#ExchangeClient.TrailingStop) places a stop
that follows the price at a fixed distance, as a percentage (`"1.5"` is 1.5%)
or an absolute price distance, and returns its `oid`. `ActivationPx` delays
trailing until the price reaches it:

```go
pct := hyperliquid.Decimal("1.5")
oid, err := ex.TrailingStop(ctx, hyperliquid.TrailingStopAction{
	Asset:       eth.ID,
	IsBuy:       false, // protects a long
	Size:        "0.01",
	ReduceOnly:  true,
	Retracement: hyperliquid.Retracement{Pct: &pct},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("trailing stop oid", oid)
```

## Leverage and margin

[`UpdateLeverage`](/docs/reference#ExchangeClient.UpdateLeverage) sets the
leverage and margin mode (cross or isolated) of a perp for the account.
[`UpdateIsolatedMargin`](/docs/reference#ExchangeClient.UpdateIsolatedMargin)
adds margin to or removes it from an isolated position; `Ntli` is the USDC
amount times 10<sup>6</sup>, negative to remove:

```go
err := ex.UpdateLeverage(ctx, hyperliquid.UpdateLeverageAction{
	Asset:    eth.ID,
	IsCross:  false,
	Leverage: 5,
})
if err != nil {
	log.Fatal(err)
}

err = ex.UpdateIsolatedMargin(ctx, hyperliquid.UpdateIsolatedMarginAction{
	Asset: eth.ID,
	IsBuy: true,       // the official SDKs always send true
	Ntli:  25_000_000, // add 25 USDC
})
if err != nil {
	log.Fatal(err)
}
```

The maximum leverage of each asset is in
[`Meta`](/docs/guides/market-data#perp-metadata-and-asset-contexts).

## Builder fees

A builder, such as a front end or a trading bot service, can charge a fee on
the orders it submits. The account first approves a maximum rate with its own
key, because
[`ApproveBuilderFee`](/docs/reference#ExchangeClient.ApproveBuilderFee) is a
user-signed action. `MaxFeeRate` is a percentage of notional:

```go
builder := hyperliquid.MustParseAddress("0x1234567890abcdef1234567890abcdef12345678")

owner, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_ACCOUNT_KEY"))
if err != nil {
	log.Fatal(err)
}
err = hyperliquid.NewExchangeClient(hyperliquid.Testnet, owner).ApproveBuilderFee(ctx, hyperliquid.ApproveBuilderFeeAction{
	Builder:    builder,
	MaxFeeRate: "0.01", // 0.01%
})
if err != nil {
	log.Fatal(err)
}
```

Orders then name the builder and the fee in
[`Builder`](/docs/reference#Builder), in tenths of a basis point (`10` is
0.01%), at most the approved rate:

```go
results, err := ex.Order(ctx, hyperliquid.OrderAction{
	Orders:  []hyperliquid.Order{order},
	Builder: &hyperliquid.Builder{Address: builder, Fee: 10},
})
```

[`InfoClient.MaxBuilderFee`](/docs/reference#InfoClient.MaxBuilderFee)
returns the rate an account has approved for a builder, in the same tenths of
a basis point.
