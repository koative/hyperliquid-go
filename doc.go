// Package hyperliquid is a client for the Hyperliquid exchange API.
//
// It has three clients:
//
//   - [InfoClient] reads market data, account state and the explorer. It
//     needs no keys.
//   - [ExchangeClient] signs and submits actions: orders, cancels,
//     transfers, vault, staking and deployer operations.
//   - [WebSocketClient] streams subscriptions, and can also carry
//     InfoClient and ExchangeClient requests (see [WithWebSocket]).
//
// Every method takes a [context.Context] and every client is safe for
// concurrent use.
//
// # Quick start
//
//	info := hyperliquid.NewInfoClient(hyperliquid.Mainnet)
//	book, err := info.L2Book(ctx, hyperliquid.L2BookRequest{Coin: "BTC"})
//
//	signer, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_AGENT_KEY"))
//	ex := hyperliquid.NewExchangeClient(hyperliquid.Mainnet, signer)
//	statuses, err := ex.Order(ctx, hyperliquid.OrderAction{Orders: []hyperliquid.Order{{
//		Asset: 0, IsBuy: true, Price: "50000", Size: "0.001",
//		Type: hyperliquid.OrderType{Limit: &hyperliquid.LimitOrder{Tif: hyperliquid.TifGtc}},
//	}}})
//
// # Networks
//
// [Mainnet] and [Testnet] hold the API URLs of each network and select the
// signature domain. Declare a [Network] value for other deployments, such as
// a local node.
//
// # Signing
//
// Actions are signed by a [Signer]. [NewPrivateKeySigner] covers in-memory
// keys; implement Signer to sign with a KMS or HSM. Trading and other L1
// actions may be signed by an API wallet approved with
// [ExchangeClient.ApproveAgent]. User-signed actions (transfers,
// withdrawals, agent and builder approvals, staking) must be signed by the
// account's own key. Signatures are tested byte for byte against the
// official Python SDK.
//
// # Assets and numbers
//
// Exchange actions identify markets by asset ID. [LoadMarkets] resolves
// names such as "BTC", "PURR/USDC", "@107" or "xyz:TSLA" (a builder-deployed
// perp) to an [Asset] with its ID and size decimals.
//
// Prices, sizes and amounts are [Decimal] strings, exactly as Hyperliquid
// sends them, so no precision is lost. Round order prices and sizes with
// [FormatPrice] and [FormatSize] (or [Asset.FormatPrice]). Timestamps are
// milliseconds since the Unix epoch.
//
// # Errors
//
// Requests rejected by the API return an [*APIError]. Batch actions such as
// [ExchangeClient.Order] and [ExchangeClient.Cancel] are accepted or
// rejected per item; failed items are reported as a joined [*StatusError]
// per item, alongside the per-item results:
//
//	statuses, err := ex.Order(ctx, action)
//	var rejected *hyperliquid.StatusError
//	if errors.As(err, &rejected) {
//		log.Printf("order %d: %s", rejected.Index, rejected.Message)
//	}
//
// # Nonces, vaults and expiry
//
// Each action is signed with a nonce: the current time in milliseconds,
// strictly increasing per [ExchangeClient]. Share one ExchangeClient per
// signer so concurrent actions never reuse a nonce. [ExchangeClient.WithVault]
// trades on behalf of a vault or sub-account, and
// [ExchangeClient.WithExpiresAfter] makes the exchange reject actions that
// arrive late.
//
// # Multi-sig
//
// For a multi-sig account, each authorized user signs the action with
// [ExchangeClient.SignMultiSig], and the leader submits it with the
// collected signatures using [ExchangeClient.MultiSig].
//
// # WebSocket
//
// [DialWebSocket] connects to the streaming API. Each subscription method
// takes a handler that receives messages in order on the subscription's own
// goroutine:
//
//	ws, err := hyperliquid.DialWebSocket(ctx, hyperliquid.Mainnet)
//	defer ws.Close()
//	sub, err := ws.Trades(ctx, hyperliquid.TradesSubscription{Coin: "BTC"}, func(trades []hyperliquid.Trade) {
//		fmt.Println(trades[0].Px)
//	})
//
// The client keeps the connection alive, reconnects with backoff and
// re-subscribes automatically.
//
// This package is not affiliated with Hyperliquid Labs.
package hyperliquid
