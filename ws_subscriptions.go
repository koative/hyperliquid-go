package hyperliquid

import (
	"context"
	"strings"
)

func matchCoin(coin string) func(*wsProbe) bool {
	return func(p *wsProbe) bool { return p.Coin == coin }
}

func matchUser(user Address) func(*wsProbe) bool {
	return func(p *wsProbe) bool { return strings.EqualFold(p.User, user.String()) }
}

func matchUserDex(user Address, dex string) func(*wsProbe) bool {
	return func(p *wsProbe) bool { return p.Dex == dex && strings.EqualFold(p.User, user.String()) }
}

// AllMidsSubscription is the request of [WebSocketClient.AllMids].
type AllMidsSubscription struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// AllMids streams the mid price of every coin.
func (c *WebSocketClient) AllMids(ctx context.Context, req AllMidsSubscription, handler func(AllMidsEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "allMids", "allMids", req,
		func(p *wsProbe) bool { return p.Dex == req.Dex }, handler)
}

// AssetCtxsSubscription is the request of [WebSocketClient.AssetCtxs].
type AssetCtxsSubscription struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex"`
}

// AssetCtxs streams the contexts of every perp asset of a dex.
func (c *WebSocketClient) AssetCtxs(ctx context.Context, req AssetCtxsSubscription, handler func(AssetCtxsEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "assetCtxs", "assetCtxs", req,
		func(p *wsProbe) bool { return p.Dex == req.Dex }, handler)
}

// AllDexsAssetCtxs streams the contexts of every perp asset of every dex.
func (c *WebSocketClient) AllDexsAssetCtxs(ctx context.Context, handler func(AllDexsAssetCtxsEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "allDexsAssetCtxs", "allDexsAssetCtxs", noParams{}, nil, handler)
}

// FastAssetCtxs streams mark and mid prices of every perp asset: a full
// snapshot, then changes. A subscription that joins a live identical one
// receives only changes.
func (c *WebSocketClient) FastAssetCtxs(ctx context.Context, handler func(FastAssetCtxs)) (*Subscription, error) {
	return subscribe(ctx, c, "fastAssetCtxs", "fastAssetCtxs", noParams{}, nil, handler)
}

// SpotAssetCtxs streams the contexts of every spot asset.
func (c *WebSocketClient) SpotAssetCtxs(ctx context.Context, handler func([]SpotAssetCtx)) (*Subscription, error) {
	return subscribe(ctx, c, "spotAssetCtxs", "spotAssetCtxs", noParams{}, nil, handler)
}

// ActiveAssetCtxSubscription is the request of
// [WebSocketClient.ActiveAssetCtx] and [WebSocketClient.ActiveSpotAssetCtx].
type ActiveAssetCtxSubscription struct {
	Coin string `json:"coin"`
}

// ActiveAssetCtx streams the context of a perp asset. Use
// [WebSocketClient.ActiveSpotAssetCtx] for spot coins such as "@107".
func (c *WebSocketClient) ActiveAssetCtx(ctx context.Context, req ActiveAssetCtxSubscription, handler func(ActiveAssetCtxEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "activeAssetCtx", "activeAssetCtx", req, matchCoin(req.Coin), handler)
}

// ActiveSpotAssetCtx streams the context of a spot asset.
func (c *WebSocketClient) ActiveSpotAssetCtx(ctx context.Context, req ActiveAssetCtxSubscription, handler func(ActiveSpotAssetCtxEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "activeAssetCtx", "activeSpotAssetCtx", req, matchCoin(req.Coin), handler)
}

// ActiveAssetDataSubscription is the request of
// [WebSocketClient.ActiveAssetData].
type ActiveAssetDataSubscription struct {
	Coin string  `json:"coin"`
	User Address `json:"user"`
}

// ActiveAssetData streams a user's leverage and trading limits on a perp
// asset.
func (c *WebSocketClient) ActiveAssetData(ctx context.Context, req ActiveAssetDataSubscription, handler func(ActiveAssetData)) (*Subscription, error) {
	return subscribe(ctx, c, "activeAssetData", "activeAssetData", req,
		func(p *wsProbe) bool { return p.Coin == req.Coin && strings.EqualFold(p.User, req.User.String()) }, handler)
}

// L2BookSubscription is the request of [WebSocketClient.L2Book].
type L2BookSubscription struct {
	Coin string `json:"coin"`
	// NSigFigs aggregates levels to 2-5 significant figures; nil means full
	// precision.
	NSigFigs *int `json:"nSigFigs"`
	// Mantissa (2 or 5) further aggregates levels when NSigFigs is 5.
	Mantissa *int `json:"mantissa"`
	// Fast streams 5 levels per side every 0.5 seconds instead of 20 levels
	// every 5 seconds.
	Fast bool `json:"fast,omitempty"`
}

// L2Book streams order book snapshots of a coin.
//
// Messages carry only the coin, so subscriptions to one coin with
// different aggregation or speed receive each other's snapshots.
func (c *WebSocketClient) L2Book(ctx context.Context, req L2BookSubscription, handler func(L2Book)) (*Subscription, error) {
	return subscribe(ctx, c, "l2Book", "l2Book", req, matchCoin(req.Coin), handler)
}

// BboSubscription is the request of [WebSocketClient.Bbo].
type BboSubscription struct {
	Coin string `json:"coin"`
}

// Bbo streams the best bid and offer of a coin on every change.
func (c *WebSocketClient) Bbo(ctx context.Context, req BboSubscription, handler func(BboEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "bbo", "bbo", req, matchCoin(req.Coin), handler)
}

// TradesSubscription is the request of [WebSocketClient.Trades].
type TradesSubscription struct {
	Coin string `json:"coin"`
}

// Trades streams the trades of a coin.
func (c *WebSocketClient) Trades(ctx context.Context, req TradesSubscription, handler func([]Trade)) (*Subscription, error) {
	return subscribe(ctx, c, "trades", "trades", req, matchCoin(req.Coin), handler)
}

// CandleSubscription is the request of [WebSocketClient.Candle].
type CandleSubscription struct {
	Coin     string         `json:"coin"`
	Interval CandleInterval `json:"interval"`
}

// Candle streams the current candle of a coin on every change.
func (c *WebSocketClient) Candle(ctx context.Context, req CandleSubscription, handler func(Candle)) (*Subscription, error) {
	return subscribe(ctx, c, "candle", "candle", req,
		func(p *wsProbe) bool { return p.S == req.Coin && p.I == string(req.Interval) }, handler)
}

// ClearinghouseStateSubscription is the request of
// [WebSocketClient.ClearinghouseState].
type ClearinghouseStateSubscription struct {
	User Address `json:"user"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex"`
}

// ClearinghouseState streams a user's perp account state on a dex.
func (c *WebSocketClient) ClearinghouseState(ctx context.Context, req ClearinghouseStateSubscription, handler func(ClearinghouseStateEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "clearinghouseState", "clearinghouseState", req, matchUserDex(req.User, req.Dex), handler)
}

// AllDexsClearinghouseStateSubscription is the request of
// [WebSocketClient.AllDexsClearinghouseState].
type AllDexsClearinghouseStateSubscription struct {
	User Address `json:"user"`
}

// AllDexsClearinghouseState streams a user's perp account state on every
// dex.
func (c *WebSocketClient) AllDexsClearinghouseState(ctx context.Context, req AllDexsClearinghouseStateSubscription, handler func(AllDexsClearinghouseStateEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "allDexsClearinghouseState", "allDexsClearinghouseState", req, matchUser(req.User), handler)
}

// SpotStateSubscription is the request of [WebSocketClient.SpotState].
type SpotStateSubscription struct {
	User Address `json:"user"`
	// IgnorePortfolioMargin reports balances without portfolio margin
	// adjustments.
	IgnorePortfolioMargin bool `json:"ignorePortfolioMargin,omitempty"`
}

// SpotState streams a user's spot balances.
func (c *WebSocketClient) SpotState(ctx context.Context, req SpotStateSubscription, handler func(SpotStateEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "spotState", "spotState", req, matchUser(req.User), handler)
}

// OpenOrdersSubscription is the request of [WebSocketClient.OpenOrders].
type OpenOrdersSubscription struct {
	User Address `json:"user"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex"`
}

// OpenOrders streams a user's open orders on a dex.
func (c *WebSocketClient) OpenOrders(ctx context.Context, req OpenOrdersSubscription, handler func(OpenOrdersEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "openOrders", "openOrders", req, matchUserDex(req.User, req.Dex), handler)
}

// TwapStatesSubscription is the request of [WebSocketClient.TwapStates].
type TwapStatesSubscription struct {
	User Address `json:"user"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex"`
}

// TwapStates streams a user's running TWAP orders on a dex.
func (c *WebSocketClient) TwapStates(ctx context.Context, req TwapStatesSubscription, handler func(TwapStatesEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "twapStates", "twapStates", req, matchUserDex(req.User, req.Dex), handler)
}

// OrderUpdatesSubscription is the request of [WebSocketClient.OrderUpdates].
type OrderUpdatesSubscription struct {
	User Address `json:"user"`
}

// OrderUpdates streams changes to a user's orders.
//
// Messages do not name the user, so subscriptions for several users on one
// client all receive every user's updates.
func (c *WebSocketClient) OrderUpdates(ctx context.Context, req OrderUpdatesSubscription, handler func([]OrderUpdate)) (*Subscription, error) {
	return subscribe(ctx, c, "orderUpdates", "orderUpdates", req, nil, handler)
}

// UserEventsSubscription is the request of [WebSocketClient.UserEvents].
type UserEventsSubscription struct {
	User Address `json:"user"`
}

// UserEvents streams a user's fills, funding payments, liquidations and
// exchange-initiated cancels.
//
// Messages do not name the user, so subscriptions for several users on one
// client all receive every user's events.
func (c *WebSocketClient) UserEvents(ctx context.Context, req UserEventsSubscription, handler func(UserEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userEvents", "user", req, nil, handler)
}

// UserFillsSubscription is the request of [WebSocketClient.UserFills].
type UserFillsSubscription struct {
	User Address `json:"user"`
	// AggregateByTime combines the partial fills of an order that matched
	// at the same time.
	AggregateByTime bool `json:"aggregateByTime"`
}

// UserFills streams a user's fills, starting with a snapshot of recent ones
// unless it joins a live identical subscription.
func (c *WebSocketClient) UserFills(ctx context.Context, req UserFillsSubscription, handler func(UserFillsEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userFills", "userFills", req, matchUser(req.User), handler)
}

// UserFundingsSubscription is the request of [WebSocketClient.UserFundings].
type UserFundingsSubscription struct {
	User Address `json:"user"`
}

// UserFundings streams a user's funding payments, starting with a snapshot
// of recent ones unless it joins a live identical subscription.
func (c *WebSocketClient) UserFundings(ctx context.Context, req UserFundingsSubscription, handler func(UserFundingsEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userFundings", "userFundings", req, matchUser(req.User), handler)
}

// UserNonFundingLedgerUpdatesSubscription is the request of
// [WebSocketClient.UserNonFundingLedgerUpdates].
type UserNonFundingLedgerUpdatesSubscription struct {
	User Address `json:"user"`
}

// UserNonFundingLedgerUpdates streams a user's deposits, withdrawals,
// transfers and other ledger changes, starting with a snapshot of recent
// ones unless it joins a live identical subscription.
func (c *WebSocketClient) UserNonFundingLedgerUpdates(ctx context.Context, req UserNonFundingLedgerUpdatesSubscription, handler func(UserNonFundingLedgerUpdatesEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userNonFundingLedgerUpdates", "userNonFundingLedgerUpdates", req, matchUser(req.User), handler)
}

// UserHistoricalOrdersSubscription is the request of
// [WebSocketClient.UserHistoricalOrders].
type UserHistoricalOrdersSubscription struct {
	User Address `json:"user"`
}

// UserHistoricalOrders streams a user's order status changes, starting with
// a snapshot of recent ones unless it joins a live identical subscription.
func (c *WebSocketClient) UserHistoricalOrders(ctx context.Context, req UserHistoricalOrdersSubscription, handler func(UserHistoricalOrdersEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userHistoricalOrders", "userHistoricalOrders", req, matchUser(req.User), handler)
}

// UserTwapHistorySubscription is the request of
// [WebSocketClient.UserTwapHistory].
type UserTwapHistorySubscription struct {
	User Address `json:"user"`
}

// UserTwapHistory streams a user's TWAP order status changes, starting with
// a snapshot of recent ones unless it joins a live identical subscription.
func (c *WebSocketClient) UserTwapHistory(ctx context.Context, req UserTwapHistorySubscription, handler func(UserTwapHistoryEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userTwapHistory", "userTwapHistory", req, matchUser(req.User), handler)
}

// UserTwapSliceFillsSubscription is the request of
// [WebSocketClient.UserTwapSliceFills].
type UserTwapSliceFillsSubscription struct {
	User Address `json:"user"`
}

// UserTwapSliceFills streams the fills of a user's TWAP orders, starting
// with a snapshot of recent ones unless it joins a live identical
// subscription.
func (c *WebSocketClient) UserTwapSliceFills(ctx context.Context, req UserTwapSliceFillsSubscription, handler func(UserTwapSliceFillsEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "userTwapSliceFills", "userTwapSliceFills", req, matchUser(req.User), handler)
}

// NotificationSubscription is the request of [WebSocketClient.Notification].
type NotificationSubscription struct {
	User Address `json:"user"`
}

// Notification streams the notifications the web app shows a user.
//
// Messages do not name the user, so subscriptions for several users on one
// client all receive every user's notifications.
func (c *WebSocketClient) Notification(ctx context.Context, req NotificationSubscription, handler func(NotificationEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "notification", "notification", req, nil, handler)
}

// WebData3Subscription is the request of [WebSocketClient.WebData3].
type WebData3Subscription struct {
	User Address `json:"user"`
}

// WebData3 streams the account data shown by the web app.
func (c *WebSocketClient) WebData3(ctx context.Context, req WebData3Subscription, handler func(WebData3)) (*Subscription, error) {
	return subscribe(ctx, c, "webData3", "webData3", req,
		func(p *wsProbe) bool { return strings.EqualFold(p.UserState.User, req.User.String()) }, handler)
}

// OutcomeMetaUpdates streams changes to outcome market metadata.
func (c *WebSocketClient) OutcomeMetaUpdates(ctx context.Context, handler func(OutcomeMetaUpdatesEvent)) (*Subscription, error) {
	return subscribe(ctx, c, "outcomeMetaUpdates", "outcomeMetaUpdates", noParams{}, nil, handler)
}

// ExplorerBlock streams new L1 blocks, without their transactions. It
// requires a client from [DialExplorerWebSocket].
func (c *WebSocketClient) ExplorerBlock(ctx context.Context, handler func([]BlockDetails)) (*Subscription, error) {
	return subscribe(ctx, c, "explorerBlock", "explorerBlock", noParams{}, nil, handler)
}

// ExplorerTxs streams new L1 transactions. It requires a client from
// [DialExplorerWebSocket].
func (c *WebSocketClient) ExplorerTxs(ctx context.Context, handler func([]ExplorerTx)) (*Subscription, error) {
	return subscribe(ctx, c, "explorerTxs", "explorerTxs", noParams{}, nil, handler)
}
