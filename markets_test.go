package hyperliquid

import (
	"math"
	"reflect"
	"testing"
)

func TestMarketsAssetIDs(t *testing.T) {
	dexs := []*PerpDex{nil, {Name: "xyz"}, {Name: "flx"}}
	metas := []Meta{
		{Universe: []PerpAssetMeta{{Name: "BTC", SzDecimals: 5}, {Name: "ETH", SzDecimals: 4}}},
		{Universe: []PerpAssetMeta{{Name: "xyz:XYZ100", SzDecimals: 4}, {Name: "xyz:TSLA", SzDecimals: 3}}},
		{Universe: []PerpAssetMeta{{Name: "flx:COIN", SzDecimals: 2}}},
	}
	spot := &SpotMeta{
		Tokens: []SpotTokenMeta{{Name: "USDC", Index: 0, SzDecimals: 8}, {Name: "PURR", Index: 1}, {Name: "HYPE", Index: 150, SzDecimals: 2}},
		Universe: []SpotPairMeta{
			{Name: "PURR/USDC", Tokens: [2]int{1, 0}, Index: 0},
			// Pair indices need not match positions.
			{Name: "@107", Tokens: [2]int{150, 0}, Index: 107},
		},
	}
	m, err := newMarkets(dexs, metas, spot)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		want Asset
	}{
		{"BTC", Asset{ID: 0, Name: "BTC", SzDecimals: 5}},
		{"ETH", Asset{ID: 1, Name: "ETH", SzDecimals: 4}},
		{"xyz:TSLA", Asset{ID: 110001, Name: "xyz:TSLA", SzDecimals: 3, Dex: "xyz"}},
		{"flx:COIN", Asset{ID: 120000, Name: "flx:COIN", SzDecimals: 2, Dex: "flx"}},
		{"PURR/USDC", Asset{ID: 10000, Name: "PURR/USDC", Spot: true}},
		{"@0", Asset{ID: 10000, Name: "PURR/USDC", Spot: true}},
		{"HYPE/USDC", Asset{ID: 10107, Name: "@107", SzDecimals: 2, Spot: true}},
		{"@107", Asset{ID: 10107, Name: "@107", SzDecimals: 2, Spot: true}},
	} {
		got, ok := m.Asset(tt.name)
		if !ok || got != tt.want {
			t.Errorf("Asset(%q) = %+v, %v; want %+v", tt.name, got, ok, tt.want)
		}
	}
	if _, ok := m.Asset("@1"); ok {
		t.Error("Asset(@1) found a nonexistent pair")
	}
	if got, err := (Asset{SzDecimals: 2}).FormatSize("1.239"); err != nil || got != "1.23" {
		t.Errorf("FormatSize = %q, %v; want 1.23", got, err)
	}

	// A dex list that disagrees with the metas must not yield wrong IDs.
	if _, err := newMarkets([]*PerpDex{nil, {Name: "flx"}, {Name: "xyz"}}, metas, spot); err == nil {
		t.Error("newMarkets accepted misaligned dexes")
	}
}

func TestAssetMarketOrder(t *testing.T) {
	perp := Asset{ID: 1, SzDecimals: 4}
	spot := Asset{ID: 10107, SzDecimals: 2, Spot: true}
	for _, tt := range []struct {
		a        Asset
		isBuy    bool
		size     Decimal
		mid      Decimal
		slippage float64
		px, sz   Decimal
	}{
		{perp, true, "0.12345", "2000", 0.05, "2100", "0.1234"},
		{perp, false, "0.12345", "2000", 0.05, "1900", "0.1234"},
		// mid*1.01 = 1.2469056: 2 decimals for this perp, 5 significant
		// figures (within 6 decimals) for this spot pair.
		{perp, true, "1", "1.23456", 0.01, "1.24", "1"},
		{spot, true, "1.239", "1.23456", 0.01, "1.2469", "1.23"},
		{spot, false, "3", "0.123456", 0.05, "0.11728", "3"},
		{perp, true, "1", "2000", 0, "2000", "1"},
	} {
		o, err := tt.a.MarketOrder(tt.isBuy, tt.size, tt.mid, tt.slippage)
		want := Order{Asset: tt.a.ID, IsBuy: tt.isBuy, Price: tt.px, Size: tt.sz, Type: OrderType{Limit: &LimitOrder{Tif: TifIoc}}}
		if err != nil || !reflect.DeepEqual(o, want) {
			t.Errorf("%+v: got %+v, %v; want %+v", tt, o, err, want)
		}
	}
	for _, tt := range []struct {
		isBuy    bool
		size     Decimal
		mid      Decimal
		slippage float64
	}{
		{true, "1", "2000", -0.01},
		{true, "1", "2000", math.NaN()},
		{false, "1", "2000", 1},
		{true, "1", "0", 0.05},
		{true, "1", "-5", 0.05},
		{true, "1", "", 0.05},
		{true, "0.00001", "2000", 0.05},
	} {
		if o, err := perp.MarketOrder(tt.isBuy, tt.size, tt.mid, tt.slippage); err == nil {
			t.Errorf("%+v: got %+v, want error", tt, o)
		}
	}
}
