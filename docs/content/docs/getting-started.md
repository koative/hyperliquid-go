---
title: Getting started
weight: 2
---

## Install

```sh
go get github.com/koative/hyperliquid-go
```

The SDK supports the two most recent Go releases (currently Go 1.26 and
newer).

## Read market data

[`InfoClient`](../reference/#InfoClient) needs no keys:

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
	fmt.Println("ETH best bid/ask:", book.Levels[0][0].Px, book.Levels[1][0].Px)
}
```

## Place an order on Testnet

1. Create a Testnet account at [app.hyperliquid-testnet.xyz](https://app.hyperliquid-testnet.xyz)
   and claim test USDC from the faucet.
2. Create an API wallet under **More → API** and copy its private key. An API
   (agent) wallet can trade for your account but cannot move funds.
3. Run:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/koative/hyperliquid-go"
)

func main() {
	ctx := context.Background()

	signer, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_AGENT_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	info := hyperliquid.NewInfoClient(hyperliquid.Testnet)
	ex := hyperliquid.NewExchangeClient(hyperliquid.Testnet, signer)

	// Resolve the market name to its asset ID and size decimals.
	markets, err := hyperliquid.LoadMarkets(ctx, info)
	if err != nil {
		log.Fatal(err)
	}
	eth, ok := markets.Asset("ETH")
	if !ok {
		log.Fatal("ETH not listed")
	}

	// Bid 5% below the mid, rounded to the market's tick and lot size.
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
		log.Fatal(err)
	}
	if r := results[0].Resting; r != nil {
		fmt.Println("resting order", r.Oid)
		if err := ex.Cancel(ctx, hyperliquid.CancelAction{
			Cancels: []hyperliquid.Cancel{{Asset: eth.ID, Oid: r.Oid}},
		}); err != nil {
			log.Fatal(err)
		}
	}
}
```

```sh
HL_AGENT_KEY=0x... go run .
```

{{< callout type="info" >}}
An agent key signs for the account that approved it. Orders go to that
account, and errors such as `User or API Wallet 0x… does not exist` mean the
key has not been approved on this network.
{{< /callout >}}

## Stream trades

```go
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
```

The client pings the server, reconnects with backoff and re-subscribes on its
own. Close it to stop every subscription.
