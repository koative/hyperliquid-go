# hyperliquid-go

[![Go Reference](https://pkg.go.dev/badge/github.com/koative/hyperliquid-go.svg)](https://pkg.go.dev/github.com/koative/hyperliquid-go)
[![CI](https://github.com/koative/hyperliquid-go/actions/workflows/ci.yml/badge.svg)](https://github.com/koative/hyperliquid-go/actions/workflows/ci.yml)
[![Live API](https://github.com/koative/hyperliquid-go/actions/workflows/live.yml/badge.svg)](https://github.com/koative/hyperliquid-go/actions/workflows/live.yml)
[![codecov](https://codecov.io/gh/koative/hyperliquid-go/graph/badge.svg)](https://codecov.io/gh/koative/hyperliquid-go)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/koative/hyperliquid-go/badge)](https://scorecard.dev/viewer/?uri=github.com/koative/hyperliquid-go)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A Go SDK for the [Hyperliquid](https://hyperliquid.xyz) API: market data,
account state, trading and transfers over HTTP, and real-time subscriptions
over WebSocket.

> [!IMPORTANT]
> This is an unofficial, community-maintained SDK. It is not affiliated with,
> endorsed by or supported by Hyperliquid Labs. Trading digital assets and
> perpetual futures carries a high risk of loss. The software is provided
> "as is", without warranty of any kind (see [LICENSE](LICENSE)); you are
> responsible for every order and transfer it signs. Test on Testnet first.

## Install

```sh
go get github.com/koative/hyperliquid-go
```

Requires Go 1.26 or newer. The package name is `hyperliquid`.

## Quickstart

### Read market data

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/koative/hyperliquid-go"
)

func main() {
	ctx := context.Background()
	info := hyperliquid.NewInfoClient(hyperliquid.Mainnet)

	mids, err := info.AllMids(ctx, hyperliquid.AllMidsRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("BTC mid:", mids["BTC"])

	book, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: "ETH"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("ETH bid/ask:", book.Levels[0][0].Px, book.Levels[1][0].Px)
}
```

### Place and cancel an order

Trade with an API (agent) wallet: a separate key that your account approves
with `ExchangeClient.ApproveAgent` (or in the Hyperliquid UI under *API*). It
can trade for the account but cannot withdraw or transfer funds.

```go
ctx := context.Background()

signer, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_AGENT_KEY"))
if err != nil {
	log.Fatal(err)
}
ex := hyperliquid.NewExchangeClient(hyperliquid.Testnet, signer)
info := hyperliquid.NewInfoClient(hyperliquid.Testnet)

// Orders take asset IDs; Markets resolves names such as "ETH", "PURR/USDC"
// or "xyz:TSLA" and knows each asset's size decimals.
markets, err := hyperliquid.LoadMarkets(ctx, info)
if err != nil {
	log.Fatal(err)
}
eth, _ := markets.Asset("ETH")

// Bid 5% below mid, rounded to the asset's tick and lot size.
mids, err := info.AllMids(ctx, hyperliquid.AllMidsRequest{})
if err != nil {
	log.Fatal(err)
}
mid, _ := mids[eth.Name].Float64()
px, err := eth.FormatPrice(hyperliquid.DecimalFromFloat(mid * 0.95))
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
		Type:  hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifGtc}},
	}},
})
if rejected, ok := errors.AsType[*hyperliquid.StatusError](err); ok {
	log.Fatalf("order rejected: %s", rejected.Message)
} else if err != nil {
	log.Fatal(err) // *hyperliquid.APIError, network error, ...
}
if results[0].Resting == nil {
	return // filled immediately
}

err = ex.Cancel(ctx, hyperliquid.CancelAction{
	Cancels: []hyperliquid.Cancel{{Asset: eth.ID, Oid: results[0].Resting.Oid}},
})
if err != nil {
	log.Fatal(err)
}
```

To trade for a vault or sub-account, use `ex.WithVault(address)`.

### Subscribe over WebSocket

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

ws, err := hyperliquid.DialWebSocket(ctx, hyperliquid.Mainnet)
if err != nil {
	log.Fatal(err)
}
defer ws.Close()

_, err = ws.Trades(ctx, hyperliquid.TradesSubscription{Coin: "BTC"}, func(trades []hyperliquid.Trade) {
	for _, t := range trades {
		fmt.Println(t.Side, t.Sz, "@", t.Px)
	}
})
if err != nil {
	log.Fatal(err)
}
<-ctx.Done() // stream until Ctrl-C
```

The client keeps the connection alive, reconnects and re-subscribes
automatically. Pass `hyperliquid.WithWebSocket(ws)` to `NewInfoClient` or
`NewExchangeClient` to send their requests over the same connection.

More examples are in the [package documentation](https://pkg.go.dev/github.com/koative/hyperliquid-go#pkg-examples).

## Features

- **Info API**: market data (mids, metadata, order books, candles, funding),
  account state (positions, orders, fills, fees, portfolio), spot, vaults,
  staking, sub-accounts and the explorer endpoints.
- **Exchange API**: orders (limit, trigger, TP/SL, TWAP, builder fees),
  modify and cancel (by order or client ID), leverage and margin, transfers
  and withdrawals, agent and builder approvals, vaults and sub-accounts,
  staking, multi-sig, and HIP-1/HIP-3 deploy actions.
- **WebSocket**: typed subscriptions for every channel, info and exchange
  requests over the socket, explorer block and transaction streams, with
  keep-alive, automatic reconnect and re-subscribe.
- Mainnet and Testnet, vaults and sub-accounts, expiring actions.
- Pluggable signing through the `Signer` interface (KMS, HSM, remote
  signers).

## Design

- **Lossless decimals.** Prices, sizes and amounts are `Decimal`, a string
  type that keeps Hyperliquid's wire form exactly. Convert with
  `Decimal.Float64` or the decimal library of your choice; round with
  `FormatPrice` and `FormatSize`. No `float64` touches money on the wire.
- **Small dependency tree.** No go-ethereum: the only dependencies are
  [decred secp256k1](https://pkg.go.dev/github.com/decred/dcrd/dcrec/secp256k1/v4),
  `golang.org/x/crypto/sha3` and
  [coder/websocket](https://github.com/coder/websocket). EIP-712 and
  MessagePack are implemented in the package.
- **Verified signing.** Signatures are tested byte for byte against golden
  vectors generated with the official
  [Python SDK](https://github.com/hyperliquid-dex/hyperliquid-python-sdk).
- **Checked against the live API every night.** Every Info endpoint and
  WebSocket channel is decoded strictly on Mainnet, and every exchange action
  is submitted to Testnet, where the exchange must recover our signer's
  address. API changes surface as an issue, usually before users hit them.
- **Idiomatic Go.** `context.Context` on every call; clients are safe for
  concurrent use; nonces are strictly increasing per process; errors are typed
  (`*APIError` for rejected requests, `*StatusError` per failed item of a
  batch).

## Documentation

- Guides and API reference: [koative.github.io/hyperliquid-go](https://koative.github.io/hyperliquid-go/)
  (concepts, guides for orders, transfers, WebSocket, vaults, staking,
  multi-sig and deployers; rebuilt from `main` on every change)
- API reference on [pkg.go.dev](https://pkg.go.dev/github.com/koative/hyperliquid-go)
- Hyperliquid API docs: [hyperliquid.gitbook.io](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api)
- Changes: [CHANGELOG.md](CHANGELOG.md)

## Versioning

The module follows [semantic versioning](https://semver.org). While it is at
`v0`, breaking changes may land in minor releases (`v0.x.0`) and are listed in
the changelog; patch releases are compatible.

The SDK supports the two most recent Go releases, matching the
[Go release policy](https://go.dev/doc/devel/release#policy). The minimum Go
version is only raised in a minor release.

## Contributing and security

Contributions are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md). Report
vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## License

[Apache License 2.0](LICENSE). Copyright 2026 Koative.
