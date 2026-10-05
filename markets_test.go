package hyperliquid

import "testing"

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
