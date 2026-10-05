---
title: Introduction
weight: 1
sidebar:
  open: true
---

`hyperliquid-go` is a Go client for the [Hyperliquid](https://hyperliquid.xyz)
API. It covers the full HTTP API (market data, account state, trading,
transfers, vaults, staking, multi-sig and deployer actions), the WebSocket
API and the explorer.

```go
import "github.com/koative/hyperliquid-go"
```

The package is named `hyperliquid` and has three clients:

| Client | Purpose | Needs a key |
|---|---|---|
| [`InfoClient`](/docs/reference#InfoClient) | Market data, account state, explorer | No |
| [`ExchangeClient`](/docs/reference#ExchangeClient) | Orders, cancels, transfers and every other signed action | Yes |
| [`WebSocketClient`](/docs/reference#WebSocketClient) | Real-time subscriptions, requests over the socket | Only to send actions |

{{< callout type="warning" >}}
This is an unofficial SDK, not affiliated with Hyperliquid Labs. Trading
carries a high risk of loss and the software comes without warranty. Try
everything on Testnet first.
{{< /callout >}}

## Where to go next

{{< cards >}}
  {{< card link="getting-started" title="Getting started" icon="play" subtitle="Install, read market data and place a first order on Testnet." >}}
  {{< card link="reference" title="API reference" icon="book-open" subtitle="Every type and method, generated from the source." >}}
{{< /cards >}}
