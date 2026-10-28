package hyperliquid

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Asset is a tradable market and the asset ID that exchange actions use for
// it.
type Asset struct {
	// ID is the asset ID: the universe index for main-dex perps,
	// 100000 + 10000*dexIndex + index for builder-dex perps and
	// 10000 + pair index for spot pairs.
	ID int
	// Name is the coin that info endpoints and subscriptions accept: the
	// perp name ("BTC", "xyz:TSLA") or the spot pair's own name ("PURR/USDC"
	// for canonical pairs, "@107" otherwise).
	Name string
	// SzDecimals is the size precision; for spot pairs, the base token's.
	SzDecimals int
	Spot       bool
	// Dex is the builder dex name, empty for main-dex perps and spot.
	Dex string
}

// FormatPrice rounds px to a valid price for a; see [FormatPrice].
func (a Asset) FormatPrice(px Decimal) (Decimal, error) {
	return FormatPrice(px, a.SzDecimals, a.Spot)
}

// FormatSize rounds sz to a valid size for a; see [FormatSize].
func (a Asset) FormatSize(sz Decimal) (Decimal, error) {
	return FormatSize(sz, a.SzDecimals)
}

// MarketOrder returns an IOC limit order for a, the way Hyperliquid market
// orders are placed: priced at mid*(1+slippage) for buys and
// mid*(1-slippage) for sells, so slippage 0.05 allows 5%. Get mid from
// [InfoClient.AllMids] with AllMidsRequest{Dex: a.Dex}. Price and size are
// rounded toward zero with [Asset.FormatPrice] and [Asset.FormatSize]; the
// Python SDK's market_open rounds the price to nearest instead, so a buy
// here may be up to one tick closer to mid and a sell one tick further.
func (a Asset) MarketOrder(isBuy bool, size, mid Decimal, slippage float64) (Order, error) {
	m, err := mid.Float64()
	if err != nil || !(m > 0) {
		return Order{}, fmt.Errorf("hyperliquid: market order: mid %q is not a positive number", string(mid))
	}
	if !(slippage >= 0) || !isBuy && slippage >= 1 {
		return Order{}, fmt.Errorf("hyperliquid: market order: invalid slippage %v", slippage)
	}
	if !isBuy {
		slippage = -slippage
	}
	px, err := a.FormatPrice(DecimalFromFloat(m * (1 + slippage)))
	if err != nil {
		return Order{}, err
	}
	sz, err := a.FormatSize(size)
	if err != nil {
		return Order{}, err
	}
	return Order{Asset: a.ID, IsBuy: isBuy, Price: px, Size: sz, Type: OrderType{Limit: &LimitOrder{Tif: TifIoc}}}, nil
}

// Markets resolves market names to [Asset]s. Build it with [LoadMarkets];
// it is a snapshot and is safe for concurrent reads.
type Markets struct {
	assets map[string]Asset
}

// LoadMarkets fetches the perp, builder dex and spot metadata and indexes
// every market.
func LoadMarkets(ctx context.Context, c *InfoClient) (*Markets, error) {
	dexs, err := c.PerpDexs(ctx)
	if err != nil {
		return nil, err
	}
	metas, err := c.AllPerpMetas(ctx)
	if err != nil {
		return nil, err
	}
	spot, err := c.SpotMeta(ctx)
	if err != nil {
		return nil, err
	}
	return newMarkets(dexs, metas, spot)
}

func newMarkets(dexs []*PerpDex, metas []Meta, spot *SpotMeta) (*Markets, error) {
	if len(dexs) != len(metas) {
		return nil, fmt.Errorf("hyperliquid: %d perp dexs but %d perp metas", len(dexs), len(metas))
	}
	m := &Markets{assets: make(map[string]Asset)}
	for i, meta := range metas {
		dex, base := "", 0
		if i > 0 {
			if dexs[i] == nil {
				return nil, fmt.Errorf("hyperliquid: perp dex %d is null", i)
			}
			dex, base = dexs[i].Name, 100000+10000*i
		}
		for j, u := range meta.Universe {
			// allPerpMetas is documented to follow perpDexs; verify it, since
			// a misalignment would send orders to the wrong asset.
			if dex != "" && !strings.HasPrefix(u.Name, dex+":") {
				return nil, fmt.Errorf("hyperliquid: perp %q listed under dex %q", u.Name, dex)
			}
			m.assets[u.Name] = Asset{ID: base + j, Name: u.Name, SzDecimals: u.SzDecimals, Dex: dex}
		}
	}
	tokens := make(map[int]SpotTokenMeta, len(spot.Tokens))
	for _, t := range spot.Tokens {
		tokens[t.Index] = t
	}
	for _, p := range spot.Universe {
		b, okb := tokens[p.Tokens[0]]
		q, okq := tokens[p.Tokens[1]]
		if !okb || !okq {
			continue
		}
		a := Asset{ID: 10000 + p.Index, Name: p.Name, SzDecimals: b.SzDecimals, Spot: true}
		// BASE/QUOTE is only a lookup key; the first pair wins a duplicate,
		// as in the official SDKs.
		key := b.Name + "/" + q.Name
		if _, dup := m.assets[key]; !dup {
			m.assets[key] = a
		}
		m.assets[p.Name] = a
		m.assets["@"+strconv.Itoa(p.Index)] = a
	}
	return m, nil
}

// Asset looks up a market by name: a perp ("BTC"), a builder-dex perp
// ("xyz:TSLA"), a spot pair ("PURR/USDC") or a spot pair's coin alias
// ("@107").
func (m *Markets) Asset(name string) (Asset, bool) {
	a, ok := m.assets[name]
	return a, ok
}
