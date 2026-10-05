---
title: WebSocket
weight: 4
---

[`WebSocketClient`](/docs/reference#WebSocketClient) streams
subscriptions over one connection and can carry `InfoClient` and
`ExchangeClient` requests too. It keeps the connection alive, reconnects and
re-subscribes on its own.

## Connect and subscribe

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/koative/hyperliquid-go"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ws, err := hyperliquid.DialWebSocket(ctx, hyperliquid.Mainnet)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	sub, err := ws.Bbo(ctx, hyperliquid.BboSubscription{Coin: "BTC"}, func(e hyperliquid.BboEvent) {
		if bid, ask := e.Bbo[0], e.Bbo[1]; bid != nil && ask != nil {
			fmt.Println(e.Coin, bid.Px, ask.Px)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	select {
	case <-ctx.Done(): // Ctrl-C
	case <-sub.Done():
		log.Fatal(sub.Err())
	}
}
```

[`DialWebSocket`](/docs/reference#DialWebSocket) returns
once the connection is open. Every subscription method takes a request
struct (when the stream has parameters) and a typed handler, and returns a
[`*Subscription`](/docs/reference#Subscription) once the
server confirms it.

{{< callout type="info" >}}
The `ctx` passed to a subscription method bounds only the subscribe
handshake, not the subscription's lifetime. Canceling it later does nothing;
end a subscription with `Unsubscribe` or `Close`.
{{< /callout >}}

## Streams

**Market data**

| Method | Handler receives | Notes |
|---|---|---|
| [`AllMids`](/docs/reference#WebSocketClient.AllMids) | `AllMidsEvent` | Mid of every coin; `Dex` selects a builder dex |
| [`AssetCtxs`](/docs/reference#WebSocketClient.AssetCtxs) | `AssetCtxsEvent` | Contexts of every perp of one dex |
| [`AllDexsAssetCtxs`](/docs/reference#WebSocketClient.AllDexsAssetCtxs) | `AllDexsAssetCtxsEvent` | Contexts of every perp of every dex |
| [`FastAssetCtxs`](/docs/reference#WebSocketClient.FastAssetCtxs) | `FastAssetCtxs` | Mark and mid prices: a full snapshot, then changes |
| [`SpotAssetCtxs`](/docs/reference#WebSocketClient.SpotAssetCtxs) | `[]SpotAssetCtx` | Contexts of every spot pair |
| [`ActiveAssetCtx`](/docs/reference#WebSocketClient.ActiveAssetCtx) | `ActiveAssetCtxEvent` | One perp |
| [`ActiveSpotAssetCtx`](/docs/reference#WebSocketClient.ActiveSpotAssetCtx) | `ActiveSpotAssetCtxEvent` | One spot pair, such as `"@107"` |
| [`L2Book`](/docs/reference#WebSocketClient.L2Book) | `L2Book` | Book snapshots, optionally aggregated or fast |
| [`Bbo`](/docs/reference#WebSocketClient.Bbo) | `BboEvent` | Best bid and offer on every change |
| [`Trades`](/docs/reference#WebSocketClient.Trades) | `[]Trade` | |
| [`Candle`](/docs/reference#WebSocketClient.Candle) | `Candle` | The current candle on every change |
| [`OutcomeMetaUpdates`](/docs/reference#WebSocketClient.OutcomeMetaUpdates) | `[]OutcomeMetaUpdate` | HIP-4 outcome metadata changes; no snapshot |

**User data** (every request takes a `User` address)

| Method | Handler receives | Notes |
|---|---|---|
| [`ClearinghouseState`](/docs/reference#WebSocketClient.ClearinghouseState) | `ClearinghouseStateEvent` | Perp account state on one dex |
| [`AllDexsClearinghouseState`](/docs/reference#WebSocketClient.AllDexsClearinghouseState) | `AllDexsClearinghouseStateEvent` | Perp account state on every dex |
| [`SpotState`](/docs/reference#WebSocketClient.SpotState) | `SpotStateEvent` | Spot balances |
| [`ActiveAssetData`](/docs/reference#WebSocketClient.ActiveAssetData) | `ActiveAssetData` | Leverage and trading limits on one perp |
| [`OpenOrders`](/docs/reference#WebSocketClient.OpenOrders) | `OpenOrdersEvent` | Open orders on one dex |
| [`TwapStates`](/docs/reference#WebSocketClient.TwapStates) | `TwapStatesEvent` | Running TWAPs on one dex |
| [`OrderUpdates`](/docs/reference#WebSocketClient.OrderUpdates) | `[]OrderUpdate` | Messages do not name the user (see below) |
| [`UserEvents`](/docs/reference#WebSocketClient.UserEvents) | `UserEvent` | Fills, funding, liquidations, exchange cancels; messages do not name the user |
| [`UserFills`](/docs/reference#WebSocketClient.UserFills) | `UserFillsEvent` | Starts with a snapshot (`IsSnapshot`) |
| [`UserFundings`](/docs/reference#WebSocketClient.UserFundings) | `UserFundingsEvent` | Starts with a snapshot |
| [`UserNonFundingLedgerUpdates`](/docs/reference#WebSocketClient.UserNonFundingLedgerUpdates) | `UserNonFundingLedgerUpdatesEvent` | Deposits, withdrawals, transfers; starts with a snapshot |
| [`UserHistoricalOrders`](/docs/reference#WebSocketClient.UserHistoricalOrders) | `UserHistoricalOrdersEvent` | Starts with a snapshot |
| [`UserTwapHistory`](/docs/reference#WebSocketClient.UserTwapHistory) | `UserTwapHistoryEvent` | Starts with a snapshot |
| [`UserTwapSliceFills`](/docs/reference#WebSocketClient.UserTwapSliceFills) | `UserTwapSliceFillsEvent` | Starts with a snapshot |
| [`Notification`](/docs/reference#WebSocketClient.Notification) | `NotificationEvent` | Web app notifications; messages do not name the user |
| [`WebData3`](/docs/reference#WebSocketClient.WebData3) | `WebData3` | Account data shown by the web app |

**Explorer** (needs a client from [`DialExplorerWebSocket`](#explorer-streams))

| Method | Handler receives |
|---|---|
| [`ExplorerBlock`](/docs/reference#WebSocketClient.ExplorerBlock) | `[]BlockDetails` (without transactions) |
| [`ExplorerTxs`](/docs/reference#WebSocketClient.ExplorerTxs) | `[]ExplorerTx` |

A user stream for one address:

```go
user := hyperliquid.MustParseAddress("0x...")
_, err := ws.UserFills(ctx, hyperliquid.UserFillsSubscription{User: user}, func(e hyperliquid.UserFillsEvent) {
	if e.IsSnapshot {
		return // recent history; skip it to see only new fills
	}
	for _, f := range e.Fills {
		fmt.Println(f.Coin, f.Side, f.Sz, "@", f.Px)
	}
})
if err != nil {
	log.Fatal(err)
}
```

{{< callout type="warning" >}}
`OrderUpdates`, `UserEvents` and `Notification` messages do not carry the
user's address, so the client cannot route them. If one client subscribes
them for several users, every handler receives every user's messages. Use a
separate `WebSocketClient` per user for these streams. Likewise, `L2Book`
messages carry only the coin: two subscriptions to the same coin with
different `NSigFigs`, `Mantissa` or `Fast` receive each other's snapshots.
{{< /callout >}}

## Handlers

- Each subscription has its own goroutine. Its handler receives messages one
  at a time, in order. Handlers of different subscriptions run concurrently,
  so guard shared state.
- Messages wait in a per-subscription queue while the handler runs. If the
  handler falls more than 4096 messages behind, that subscription (only)
  ends with
  [`ErrSubscriptionOverflow`](/docs/reference#ErrSubscriptionOverflow).
  Keep handlers fast; hand slow work to your own goroutine or channel.
- A message that fails to decode is skipped and reported to the
  [error handler](#errors-and-the-error-handler).

## Ending a subscription

A `Subscription` ends when you unsubscribe, when the client is closed, when
its handler overflows, or when the server rejects it while re-subscribing
after a reconnect.

```go
sub, err := ws.Trades(ctx, hyperliquid.TradesSubscription{Coin: "ETH"}, func(trades []hyperliquid.Trade) {
	fmt.Println(len(trades), "trades")
})
if err != nil {
	log.Fatal(err)
}

// Later:
if err := sub.Unsubscribe(ctx); err != nil {
	log.Print(err)
}
<-sub.Done()           // closed once the subscription has ended
fmt.Println(sub.Err()) // nil after Unsubscribe
```

[`Unsubscribe`](/docs/reference#Subscription.Unsubscribe)
waits for the server's confirmation (bounded by `ctx`). After it returns the
handler is not called again, except for a call already in progress.
[`Err`](/docs/reference#Subscription.Err) reports why a
subscription ended: `nil` while live or after `Unsubscribe`,
[`ErrWebSocketClosed`](/docs/reference#ErrWebSocketClosed)
after `Close`, `ErrSubscriptionOverflow`, or the server's `*APIError`.

[`Close`](/docs/reference#WebSocketClient.Close) ends every
subscription with `ErrWebSocketClosed` and returns once the client's
goroutines have exited (handlers still running finish on their own). Later
calls on the client return `ErrWebSocketClosed`.

## Shared subscriptions

Subscriptions with an identical request share one server subscription; each
gets its own handler and queue, and the server subscription is removed when
the last one unsubscribes. A subscription that joins a live one does not
receive the stream's initial snapshot (the `UserFills` history, the first
full frame of `FastAssetCtxs`), only later messages. If a second consumer
needs the snapshot, fetch it with the matching `InfoClient` method or use a
separate `WebSocketClient`.

## Reconnects

The client pings the server every 30 seconds. When the connection drops or
a ping goes unanswered, it reconnects with capped exponential backoff and
jitter (at most 30 seconds between attempts), then re-subscribes every live
subscription.
Your handlers keep running; nothing needs to be re-registered.

- Messages published while disconnected are not replayed. Streams that
  start with a snapshot may deliver another snapshot after re-subscribing,
  so treat `IsSnapshot` messages as state to reconcile, not as new events.
- Requests in flight when the connection drops (see
  [requests over the socket](#requests-over-the-socket)) fail with
  [`ErrWebSocketDisconnected`](/docs/reference#ErrWebSocketDisconnected).
  The request may or may not have executed: check order state (by client
  order ID, for example) before retrying an action. Requests made while
  reconnecting wait for the new connection.

## Errors and the error handler

Errors that belong to no call (lost connections, failed reconnects, server
errors matching no request, undecodable messages) are discarded unless you
pass
[`WithWebSocketErrorHandler`](/docs/reference#WithWebSocketErrorHandler):

```go
ws, err := hyperliquid.DialWebSocket(ctx, hyperliquid.Mainnet,
	hyperliquid.WithWebSocketErrorHandler(func(err error) {
		if errors.Is(err, hyperliquid.ErrWebSocketDisconnected) {
			slog.Warn("websocket disconnected, reconnecting", "err", err)
			return
		}
		slog.Error("websocket", "err", err)
	}),
)
if err != nil {
	log.Fatal(err)
}
defer ws.Close()
```

The handler runs on the client's internal goroutines, possibly concurrently.
It must return promptly and must not call `Close`, subscribe, or send
requests through the client; signal another goroutine if you need to react.

## Limits

The server limits each connection, and the client enforces the limits
itself because the server's rejection does not say which request it refers
to. Subscribing past a limit returns an error:

- 1000 subscriptions per client (identical requests count once).
- 15 distinct users across user streams per client. Hyperliquid's
  [rate limits page](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/rate-limits-and-user-limits)
  documents 10; mainnet accepts 15.

Spread more users over several clients.

## Requests over the socket

[`WithWebSocket`](/docs/reference#WithWebSocket) sends the
requests of an `InfoClient` or `ExchangeClient` as WebSocket post requests
over an existing connection instead of HTTP:

```go
ws, err := hyperliquid.DialWebSocket(ctx, hyperliquid.Mainnet)
if err != nil {
	log.Fatal(err)
}
defer ws.Close()

info := hyperliquid.NewInfoClient(hyperliquid.Mainnet, hyperliquid.WithWebSocket(ws))
ex := hyperliquid.NewExchangeClient(hyperliquid.Mainnet, signer, hyperliquid.WithWebSocket(ws))

book, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: "BTC"})
if err != nil {
	log.Fatal(err)
}
fmt.Println(book.Levels[0][0].Px)
_ = ex // ex.Order, ex.Cancel, ... now go over ws
```

The methods and their results are the same as over HTTP. Explorer requests
(`BlockDetails`, `TxDetails`, ...) still use HTTP. After `ws.Close()`, these
clients fail with `ErrWebSocketClosed`.

## Explorer streams

The explorer has its own WebSocket.
[`DialExplorerWebSocket`](/docs/reference#DialExplorerWebSocket)
connects to it; such a client serves only `ExplorerBlock` and `ExplorerTxs`.

```go
explorer, err := hyperliquid.DialExplorerWebSocket(ctx, hyperliquid.Mainnet)
if err != nil {
	log.Fatal(err)
}
defer explorer.Close()

_, err = explorer.ExplorerBlock(ctx, func(blocks []hyperliquid.BlockDetails) {
	for _, b := range blocks {
		fmt.Println("block", b.Height, b.NumTxs, "txs")
	}
})
if err != nil {
	log.Fatal(err)
}
_, err = explorer.ExplorerTxs(ctx, func(txs []hyperliquid.ExplorerTx) {
	for _, tx := range txs {
		fmt.Println(tx.Hash, tx.User, string(tx.Action))
	}
})
if err != nil {
	log.Fatal(err)
}
```
