package hyperliquid

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDecimalMarshalNormalizes(t *testing.T) {
	for in, want := range map[Decimal]string{
		"00123": `"123"`, "1.2000": `"1.2"`, ".5": `"0.5"`, "-0.0": `"0"`, "5.": `"5"`,
		"-1.50": `"-1.5"`, "40000.0": `"40000"`, "0": `"0"`, "": `""`,
	} {
		got, err := json.Marshal(in)
		if err != nil || string(got) != want {
			t.Errorf("Marshal(%q) = %s, %v; want %s", in, got, err, want)
		}
	}
	for _, bad := range []Decimal{"1e5", "abc", "1.2.3", "+1", "-", ".", " 1", "NaN"} {
		if _, err := json.Marshal(bad); err == nil {
			t.Errorf("Marshal(%q): want error", bad)
		}
	}
}

func TestDecimalUnmarshal(t *testing.T) {
	var v struct{ A, B, C Decimal }
	if err := json.Unmarshal([]byte(`{"A":"1.50","B":2.25,"C":null}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.A != "1.50" || v.B != "2.25" || v.C != "" {
		t.Errorf("got %+v", v)
	}
}

func TestFormatPrice(t *testing.T) {
	for _, tc := range []struct {
		px         Decimal
		szDecimals int
		spot       bool
		want       Decimal
	}{
		{"97123.456789", 0, false, "97123"},
		{"123456.7", 0, false, "123456"}, // integer part beyond 5 sig figs is kept
		{"1.23456789", 5, false, "1.2"},
		{"0.0000123456789", 0, true, "0.00001234"},
		{"0.00123456789", 0, false, "0.001234"},
		{"1670.1", 4, false, "1670.1"},
		{"1670.19", 4, false, "1670.1"},
		{"12.3456", 0, false, "12.345"},
		{"100", 2, false, "100"},
		{"-12.3456", 0, false, "-12.345"},
	} {
		got, err := FormatPrice(tc.px, tc.szDecimals, tc.spot)
		if err != nil || got != tc.want {
			t.Errorf("FormatPrice(%s, %d, %v) = %s, %v; want %s", tc.px, tc.szDecimals, tc.spot, got, err, tc.want)
		}
	}
	if _, err := FormatPrice("0.0000001", 0, false); !errors.Is(err, ErrTruncatedToZero) {
		t.Errorf("tiny price: got %v", err)
	}
}

func TestFormatSize(t *testing.T) {
	for _, tc := range []struct {
		sz         Decimal
		szDecimals int
		want       Decimal
	}{
		{"1.23456789", 5, "1.23456"}, {"0.123456789", 2, "0.12"}, {"100", 0, "100"}, {"1.999", 0, "1"},
	} {
		got, err := FormatSize(tc.sz, tc.szDecimals)
		if err != nil || got != tc.want {
			t.Errorf("FormatSize(%s, %d) = %s, %v; want %s", tc.sz, tc.szDecimals, got, err, tc.want)
		}
	}
	if _, err := FormatSize("0.009", 2); !errors.Is(err, ErrTruncatedToZero) {
		t.Errorf("tiny size: got %v", err)
	}
}

func FuzzFormatPrice(f *testing.F) {
	f.Add("97123.456789", 0, false)
	f.Fuzz(func(t *testing.T, px string, szDecimals int, spot bool) {
		got, err := FormatPrice(Decimal(px), szDecimals%10, spot)
		if err != nil {
			return
		}
		if _, ok := normalizeDecimal(string(got)); !ok {
			t.Fatalf("FormatPrice(%q) = %q, not a decimal", px, got)
		}
	})
}
