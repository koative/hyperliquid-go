---
title: Errors
weight: 3
---

Errors from the SDK fall into four groups: the API rejected the request
([`*APIError`](/docs/reference#APIError)), the exchange
accepted a batch but rejected some of its items
([`*StatusError`](/docs/reference#StatusError)), the
WebSocket client failed or ended a request or subscription
(`ErrWebSocket…` values), or the request did not complete at all (network
errors, context cancellation). For actions, the group tells you whether the
action executed.

## APIError

An `*APIError` means the API refused the request:

```go
type APIError struct {
	StatusCode int    // HTTP status code; 0 for errors reported in an ok response or over WebSocket
	Message    string // the server's message
}
```

`StatusCode` tells where the error came from:

- **Non-zero:** the HTTP response was not a `200 OK` with a JSON body, for
  example an HTTP error from the API or from a proxy in between. `Message` is
  the response body. A `200` status with a body that is not JSON is reported
  the same way.
- **Zero:** the request reached the exchange and it answered with an error,
  either as `{"status":"err"}` in an `/exchange` response or as an error over
  the WebSocket. A typical message is `User or API Wallet 0x… does not exist`:
  the signer is not approved for any account on this network.

Match it with [`errors.AsType`](https://pkg.go.dev/errors#AsType) (Go 1.26)
or `errors.As`:

```go
_, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: "BTC"})
if apiErr, ok := errors.AsType[*hyperliquid.APIError](err); ok {
	log.Printf("rejected by the API (HTTP %d): %s", apiErr.StatusCode, apiErr.Message)
} else if err != nil {
	log.Printf("request failed: %v", err) // network error, timeout, decode error
}
```

Responses that cannot be decoded return a plain wrapped error such as
`hyperliquid: decode l2Book response: …`.

## StatusError

Orders, cancels and batch modifies are batch actions: the exchange accepts the
batch and then accepts or rejects each item on its own. A rejected item is
reported as a `*StatusError` with its position in the request:

```go
type StatusError struct {
	Index   int    // position of the failed item in the request
	Message string // the exchange's reason
}
```

[`ExchangeClient.Order`](/docs/reference#ExchangeClient.Order),
[`BatchModify`](/docs/reference#ExchangeClient.BatchModify),
[`Cancel`](/docs/reference#ExchangeClient.Cancel),
[`CancelByCloid`](/docs/reference#ExchangeClient.CancelByCloid)
and [`MultiSig`](/docs/reference#ExchangeClient.MultiSig)
(when the inner action is a batch) join one `*StatusError` per failed item
with [`errors.Join`](https://pkg.go.dev/errors#Join). `Order` and
`BatchModify` return the per-item results **together with** the error, so the
items that succeeded are not lost:

```go
results, err := ex.Order(ctx, action)
var apiErr *hyperliquid.APIError
if errors.As(err, &apiErr) {
	log.Fatalf("whole action rejected: %s", apiErr.Message) // nothing was placed
}
for i, r := range results {
	switch {
	case r.Error != "":
		log.Printf("order %d rejected: %s", i, r.Error)
	case r.Resting != nil:
		log.Printf("order %d resting: oid %d", i, r.Resting.Oid)
	case r.Filled != nil:
		log.Printf("order %d filled %s @ %s", i, r.Filled.TotalSz, r.Filled.AvgPx)
	}
}
```

`errors.As` and `errors.AsType` return only the **first** `*StatusError` in
a joined error. To visit every rejected item without the results, as for
`Cancel`, unwrap the join:

```go
err := ex.Cancel(ctx, hyperliquid.CancelAction{Cancels: cancels})
if joined, ok := err.(interface{ Unwrap() []error }); ok {
	for _, e := range joined.Unwrap() {
		if se, ok := e.(*hyperliquid.StatusError); ok {
			log.Printf("cancel %d failed: %s", se.Index, se.Message) // e.g. already filled
		}
	}
} else if err != nil {
	log.Fatal(err)
}
```

[`TwapOrder`](/docs/reference#ExchangeClient.TwapOrder) and
[`TwapCancel`](/docs/reference#ExchangeClient.TwapCancel)
handle a single item and return a bare `*StatusError` with `Index` 0.

## WebSocket errors

The WebSocket client defines three sentinel errors. Test them with
`errors.Is`, since some are wrapped:

| Error | Returned by | Meaning |
|---|---|---|
| [`ErrWebSocketClosed`](/docs/reference#ErrWebSocketClosed) | any method after `Close`, `Subscription.Err` | The client was closed. Final. |
| [`ErrWebSocketDisconnected`](/docs/reference#ErrWebSocketDisconnected) | requests in flight when the connection dropped | The connection was lost while waiting for the response. The client reconnects on its own. |
| [`ErrSubscriptionOverflow`](/docs/reference#ErrSubscriptionOverflow) | `Subscription.Err` | The handler fell more than 4096 messages behind and the subscription was ended. Subscribe again, and keep handlers fast. |

A subscription can also end with the server's `*APIError` if the server
rejects it when it is restored after a reconnect. Watch
[`Subscription.Done`](/docs/reference#Subscription.Done):

```go
sub, err := ws.Trades(ctx, hyperliquid.TradesSubscription{Coin: "BTC"}, func(trades []hyperliquid.Trade) {
	// handle trades
})
if err != nil {
	log.Fatal(err)
}
select {
case <-sub.Done():
	if errors.Is(sub.Err(), hyperliquid.ErrSubscriptionOverflow) {
		log.Print("handler too slow; resubscribing")
	}
case <-ctx.Done():
}
```

Errors that belong to no request, such as a lost connection (wrapping
`ErrWebSocketDisconnected`) or a failed reconnect, are discarded unless you
install a handler with
[`WithWebSocketErrorHandler`](/docs/reference#WithWebSocketErrorHandler):

```go
ws, err := hyperliquid.DialWebSocket(ctx, hyperliquid.Mainnet,
	hyperliquid.WithWebSocketErrorHandler(func(err error) {
		log.Printf("websocket: %v", err) // must return quickly
	}))
if err != nil {
	log.Fatal(err)
}
defer ws.Close()
```

## Did the action execute?

The error alone does not always tell. An action that left the process but
whose response never arrived may or may not have been executed:

| Result | Executed? |
|---|---|
| `nil` | Yes. |
| `*StatusError` (joined or bare) | The action was processed; the listed items were rejected and the others took effect. |
| `*APIError` with `StatusCode` 0 | No, the exchange rejected the whole action. |
| `*APIError` with an HTTP 4xx status | No, the request was refused before processing. |
| `*APIError` with an HTTP 5xx status, from the API or a proxy | Unknown. |
| `ErrWebSocketDisconnected` (with `WithWebSocket`) | Unknown: the request may or may not have been executed. |
| Context deadline or cancellation, client timeout, connection reset | Unknown once the request was sent. The default HTTP client times out after 10 seconds. |
| Signing or encoding error (invalid `Decimal`, KMS failure) | No, nothing was sent. |

When the outcome is unknown, do not resend blindly: every call is signed with
a new nonce, so a resent order is a new order. Check the state first.

Two tools make the check reliable for orders. Give every order a
[client order ID](/docs/guides/orders#client-order-ids) so
you can look it up, and send it with
[`WithExpiresAfter`](/docs/concepts/nonces#expiring-actions) so it cannot
execute late. Once the expiry has passed, an order the exchange does not know
will never execute:

```go
expiry := time.Now().Add(5 * time.Second)
order.Cloid = &cloid
_, err := ex.WithExpiresAfter(expiry).Order(ctx, hyperliquid.OrderAction{Orders: []hyperliquid.Order{order}})
if _, rejected := errors.AsType[*hyperliquid.StatusError](err); err != nil && !rejected {
	// The outcome may be unknown: wait out the expiry, then look the order up.
	time.Sleep(time.Until(expiry) + time.Second)
	status, err := info.OrderStatus(ctx, hyperliquid.OrderStatusRequest{
		User: user, // the account, not the agent
		Oid:  hyperliquid.OrderRef{Cloid: &cloid},
	})
	if err != nil {
		log.Fatal(err)
	}
	if status == nil {
		fmt.Println("order never executed; safe to resend")
	} else {
		fmt.Println("order exists with status", status.Status)
	}
}
```
