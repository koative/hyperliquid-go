package hyperliquid

import "context"

// AllMidsRequest is the request for [InfoClient.AllMids].
type AllMidsRequest struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// AllMids returns the mid price of every coin, keyed by coin name.
func (c *InfoClient) AllMids(ctx context.Context, req AllMidsRequest) (map[string]Decimal, error) {
	return infoRequest[map[string]Decimal](ctx, c, "allMids", req)
}

// MetaRequest is the request for [InfoClient.Meta].
type MetaRequest struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// Meta describes the perpetual markets of a dex.
type Meta struct {
	// Universe lists the perpetual assets. An asset's index in Universe is
	// its asset ID on the main dex; see [Markets] for builder dexes.
	Universe []PerpAssetMeta `json:"universe"`
	// MarginTables maps margin table IDs, referenced by
	// [PerpAssetMeta.MarginTableID], to their tables.
	MarginTables TupleMap[int, MarginTable] `json:"marginTables"`
	// CollateralToken is the spot token index of the dex's collateral.
	CollateralToken int `json:"collateralToken"`
}

// PerpAssetMeta describes a perpetual asset.
type PerpAssetMeta struct {
	Name          string `json:"name"`
	SzDecimals    int    `json:"szDecimals"`
	MaxLeverage   int    `json:"maxLeverage"`
	MarginTableID int    `json:"marginTableId"`
	// OnlyIsolated is deprecated upstream in favor of MarginMode.
	OnlyIsolated bool `json:"onlyIsolated,omitempty"`
	IsDelisted   bool `json:"isDelisted,omitempty"`
	// MarginMode is "strictIsolated", "noCross" or empty.
	MarginMode string `json:"marginMode,omitempty"`
	// GrowthMode is "enabled" or empty.
	GrowthMode string `json:"growthMode,omitempty"`
	// DeployerFeeScale is the builder dex deployer's fee scale; empty on the
	// main dex.
	DeployerFeeScale Decimal `json:"deployerFeeScale,omitempty"`
	// LastFeeScaleChangeTime is an ISO 8601 time without zone, such as
	// "2025-11-23T17:37:10.033211662"; empty on the main dex.
	LastFeeScaleChangeTime string `json:"lastFeeScaleChangeTime,omitempty"`
}

// MarginTable defines leverage tiers by position notional.
type MarginTable struct {
	Description string       `json:"description"`
	MarginTiers []MarginTier `json:"marginTiers"`
}

// MarginTier is one tier of a [MarginTable].
type MarginTier struct {
	LowerBound  Decimal `json:"lowerBound"`
	MaxLeverage int     `json:"maxLeverage"`
}

// Meta returns perpetual market metadata.
func (c *InfoClient) Meta(ctx context.Context, req MetaRequest) (*Meta, error) {
	return infoRequest[*Meta](ctx, c, "meta", req)
}

// AllPerpMetas returns the [Meta] of every perp dex, indexed like
// [InfoClient.PerpDexs]: element 0 is the main dex.
func (c *InfoClient) AllPerpMetas(ctx context.Context) ([]Meta, error) {
	return infoRequest[[]Meta](ctx, c, "allPerpMetas", noParams{})
}

// PerpAssetCtx is the live market state of a perpetual asset.
type PerpAssetCtx struct {
	PrevDayPx    Decimal  `json:"prevDayPx"`
	DayNtlVlm    Decimal  `json:"dayNtlVlm"`
	MarkPx       Decimal  `json:"markPx"`
	MidPx        *Decimal `json:"midPx"`
	Funding      Decimal  `json:"funding"`
	OpenInterest Decimal  `json:"openInterest"`
	Premium      *Decimal `json:"premium"`
	OraclePx     Decimal  `json:"oraclePx"`
	// ImpactPxs holds the bid and ask impact prices, or nil.
	ImpactPxs  *[2]Decimal `json:"impactPxs"`
	DayBaseVlm Decimal     `json:"dayBaseVlm"`
}

// MetaAndAssetCtxsRequest is the request for [InfoClient.MetaAndAssetCtxs].
type MetaAndAssetCtxsRequest struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// MetaAndAssetCtxs is perpetual metadata with the market state of each asset.
type MetaAndAssetCtxs struct {
	Meta Meta
	// AssetCtxs is parallel to Meta.Universe.
	AssetCtxs []PerpAssetCtx
}

// UnmarshalJSON decodes the wire form [meta, assetCtxs].
func (m *MetaAndAssetCtxs) UnmarshalJSON(b []byte) error {
	return unmarshalTuple(b, &m.Meta, &m.AssetCtxs)
}

// MetaAndAssetCtxs returns perpetual metadata and asset contexts.
func (c *InfoClient) MetaAndAssetCtxs(ctx context.Context, req MetaAndAssetCtxsRequest) (*MetaAndAssetCtxs, error) {
	return infoRequest[*MetaAndAssetCtxs](ctx, c, "metaAndAssetCtxs", req)
}

// CandleInterval is a candle duration.
type CandleInterval string

// Candle intervals.
const (
	Candle1m  CandleInterval = "1m"
	Candle3m  CandleInterval = "3m"
	Candle5m  CandleInterval = "5m"
	Candle15m CandleInterval = "15m"
	Candle30m CandleInterval = "30m"
	Candle1h  CandleInterval = "1h"
	Candle2h  CandleInterval = "2h"
	Candle4h  CandleInterval = "4h"
	Candle8h  CandleInterval = "8h"
	Candle12h CandleInterval = "12h"
	Candle1d  CandleInterval = "1d"
	Candle3d  CandleInterval = "3d"
	Candle1w  CandleInterval = "1w"
	Candle1M  CandleInterval = "1M"
)

// CandleSnapshotRequest is the request for [InfoClient.CandleSnapshot].
type CandleSnapshotRequest struct {
	Coin     string         `json:"coin"`
	Interval CandleInterval `json:"interval"`
	// StartTime and EndTime bound the candle open times in milliseconds
	// since the Unix epoch. A zero EndTime means now.
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// Candle is an OHLCV candle.
type Candle struct {
	// OpenTime and CloseTime are in milliseconds since the Unix epoch.
	OpenTime  int64          `json:"t"`
	CloseTime int64          `json:"T"`
	Coin      string         `json:"s"`
	Interval  CandleInterval `json:"i"`
	Open      Decimal        `json:"o"`
	Close     Decimal        `json:"c"`
	High      Decimal        `json:"h"`
	Low       Decimal        `json:"l"`
	// Volume is in base units.
	Volume Decimal `json:"v"`
	// Trades is the number of trades.
	Trades int `json:"n"`
}

// CandleSnapshot returns candles in a time range. At most the 5000 most
// recent candles are available.
func (c *InfoClient) CandleSnapshot(ctx context.Context, req CandleSnapshotRequest) ([]Candle, error) {
	return infoRequest[[]Candle](ctx, c, "candleSnapshot", struct {
		Req CandleSnapshotRequest `json:"req"`
	}{req})
}

// RecentTradesRequest is the request for [InfoClient.RecentTrades].
type RecentTradesRequest struct {
	Coin string `json:"coin"`
}

// Trade is a public trade.
type Trade struct {
	Coin string `json:"coin"`
	// Side is the aggressor's side.
	Side Side    `json:"side"`
	Px   Decimal `json:"px"`
	Sz   Decimal `json:"sz"`
	// Time is in milliseconds since the Unix epoch.
	Time int64  `json:"time"`
	Hash string `json:"hash"`
	// Tid is the trade ID.
	Tid int64 `json:"tid"`
	// Users holds the buyer and the seller.
	Users [2]Address `json:"users"`
}

// RecentTrades returns the most recent trades of a coin.
func (c *InfoClient) RecentTrades(ctx context.Context, req RecentTradesRequest) ([]Trade, error) {
	return infoRequest[[]Trade](ctx, c, "recentTrades", req)
}

// FundingHistoryRequest is the request for [InfoClient.FundingHistory].
type FundingHistoryRequest struct {
	Coin string `json:"coin"`
	// StartTime and EndTime are in milliseconds since the Unix epoch. A zero
	// EndTime means now.
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// FundingRate is a historical funding rate of a perpetual asset.
type FundingRate struct {
	Coin        string  `json:"coin"`
	FundingRate Decimal `json:"fundingRate"`
	Premium     Decimal `json:"premium"`
	// Time is in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
}

// FundingHistory returns the funding rates of a coin, oldest first, at most
// 500 per call.
func (c *InfoClient) FundingHistory(ctx context.Context, req FundingHistoryRequest) ([]FundingRate, error) {
	return infoRequest[[]FundingRate](ctx, c, "fundingHistory", req)
}

// PredictedFunding is the next funding rate of a coin on one venue.
type PredictedFunding struct {
	FundingRate Decimal `json:"fundingRate"`
	// NextFundingTime is in milliseconds since the Unix epoch.
	NextFundingTime int64 `json:"nextFundingTime"`
	// FundingIntervalHours is 0 when the venue does not report it.
	FundingIntervalHours int `json:"fundingIntervalHours,omitempty"`
}

// PredictedFundings returns predicted funding rates keyed by coin, then by
// venue ("HlPerp", "BinPerp", "BybitPerp", ...). A nil value means the venue
// has no prediction for the coin.
func (c *InfoClient) PredictedFundings(ctx context.Context) (TupleMap[string, TupleMap[string, *PredictedFunding]], error) {
	return infoRequest[TupleMap[string, TupleMap[string, *PredictedFunding]]](ctx, c, "predictedFundings", noParams{})
}

// PerpsAtOpenInterestCapRequest is the request for
// [InfoClient.PerpsAtOpenInterestCap].
type PerpsAtOpenInterestCapRequest struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// PerpsAtOpenInterestCap returns the coins whose open interest is at its cap.
func (c *InfoClient) PerpsAtOpenInterestCap(ctx context.Context, req PerpsAtOpenInterestCapRequest) ([]string, error) {
	return infoRequest[[]string](ctx, c, "perpsAtOpenInterestCap", req)
}

// MaxMarketOrderNtls returns the maximum market order notional keyed by
// asset max leverage.
func (c *InfoClient) MaxMarketOrderNtls(ctx context.Context) (TupleMap[int, Decimal], error) {
	return infoRequest[TupleMap[int, Decimal]](ctx, c, "maxMarketOrderNtls", noParams{})
}

// MarginTableRequest is the request for [InfoClient.MarginTable].
type MarginTableRequest struct {
	ID int `json:"id"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// MarginTable returns a margin table. Unknown IDs fail with an [*APIError].
func (c *InfoClient) MarginTable(ctx context.Context, req MarginTableRequest) (*MarginTable, error) {
	return infoRequest[*MarginTable](ctx, c, "marginTable", req)
}

// PerpCategories returns the category ("crypto", "stocks", ...) of every
// categorized perpetual, keyed by coin.
func (c *InfoClient) PerpCategories(ctx context.Context) (TupleMap[string, string], error) {
	return infoRequest[TupleMap[string, string]](ctx, c, "perpCategories", noParams{})
}

// PerpAnnotationRequest is the request for [InfoClient.PerpAnnotation].
type PerpAnnotationRequest struct {
	Coin string `json:"coin"`
}

// PerpAnnotation is descriptive metadata of a perpetual.
type PerpAnnotation struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	DisplayName string `json:"displayName,omitempty"`
	// Keywords are search terms.
	Keywords []string `json:"keywords,omitempty"`
}

// PerpAnnotation returns a perpetual's annotation, or nil if it has none.
func (c *InfoClient) PerpAnnotation(ctx context.Context, req PerpAnnotationRequest) (*PerpAnnotation, error) {
	return infoRequest[*PerpAnnotation](ctx, c, "perpAnnotation", req)
}

// PerpConciseAnnotation is a [PerpAnnotation] without the description.
type PerpConciseAnnotation struct {
	Category    string   `json:"category"`
	DisplayName string   `json:"displayName,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
}

// PerpConciseAnnotations returns the annotation of every annotated
// perpetual, keyed by coin.
func (c *InfoClient) PerpConciseAnnotations(ctx context.Context) (TupleMap[string, PerpConciseAnnotation], error) {
	return infoRequest[TupleMap[string, PerpConciseAnnotation]](ctx, c, "perpConciseAnnotations", noParams{})
}

// LiquidatablePosition is a position eligible for liquidation.
type LiquidatablePosition struct {
	User          Address       `json:"user"`
	PositionIndex PositionIndex `json:"positionIndex"`
	// MarginAvailable holds two margin figures whose meaning is undocumented.
	MarginAvailable [2]Decimal `json:"marginAvailable"`
}

// PositionIndex locates a position. Only isolated positions have been
// observed.
type PositionIndex struct {
	Isolated *IsolatedPositionIndex `json:"isolated,omitempty"`
}

// IsolatedPositionIndex locates an isolated-margin position.
type IsolatedPositionIndex struct {
	Asset int `json:"asset"`
}

// Liquidatable returns the positions currently eligible for liquidation.
func (c *InfoClient) Liquidatable(ctx context.Context) ([]LiquidatablePosition, error) {
	return infoRequest[[]LiquidatablePosition](ctx, c, "liquidatable", noParams{})
}

// L2BookRequest is the request for [InfoClient.L2Book].
type L2BookRequest struct {
	Coin string `json:"coin"`
	// NSigFigs aggregates levels to 2-5 significant figures; nil means full
	// precision.
	NSigFigs *int `json:"nSigFigs,omitempty"`
	// Mantissa (2 or 5) further aggregates levels when NSigFigs is 5.
	Mantissa *int `json:"mantissa,omitempty"`
}

// L2Book is an order book snapshot.
type L2Book struct {
	Coin string `json:"coin"`
	// Time is the snapshot time in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
	// Levels holds the bids (Levels[0]) and asks (Levels[1]), best first.
	Levels [2][]L2Level `json:"levels"`
	// Spread is set only when NSigFigs was requested.
	Spread *Decimal `json:"spread,omitempty"`
	// Fast is set on snapshots from a WebSocket subscription in fast mode.
	Fast bool `json:"fast,omitempty"`
}

// L2Level is an aggregated price level.
type L2Level struct {
	Px Decimal `json:"px"`
	Sz Decimal `json:"sz"`
	// N is the number of orders at this level.
	N int `json:"n"`
}

// L2Book returns an order book snapshot of at most 20 levels per side, or
// nil if the coin does not exist.
func (c *InfoClient) L2Book(ctx context.Context, req L2BookRequest) (*L2Book, error) {
	return infoRequest[*L2Book](ctx, c, "l2Book", req)
}
