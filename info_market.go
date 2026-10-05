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
	// its asset ID on the main dex.
	Universe []PerpAssetMeta `json:"universe"`
	// MarginTables lists the margin tables referenced by
	// [PerpAssetMeta.MarginTableID].
	MarginTables []MarginTableEntry `json:"marginTables"`
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
	GrowthMode               string `json:"growthMode,omitempty"`
	LastGrowthModeChangeTime string `json:"lastGrowthModeChangeTime,omitempty"`
}

// MarginTableEntry is a margin table with its ID.
type MarginTableEntry struct {
	ID    int
	Table MarginTable
}

// UnmarshalJSON decodes the wire form [id, table].
func (e *MarginTableEntry) UnmarshalJSON(b []byte) error {
	return unmarshalTuple(b, &e.ID, &e.Table)
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
