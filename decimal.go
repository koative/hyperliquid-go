package hyperliquid

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Decimal is a base-10 number in Hyperliquid's string wire form, such as
// "0.001" or "-12.5".
//
// Hyperliquid transmits prices, sizes and amounts as decimal strings, and
// Decimal keeps them lossless: use [Decimal.Float64] or your preferred decimal
// library for arithmetic. When marshalled to JSON a Decimal is normalized
// (leading and trailing zeros stripped, "-0" collapsed to "0"); signatures are
// computed over the normalized form, which is what the exchange expects.
type Decimal string

// ParseDecimal validates s and returns it in canonical form.
func ParseDecimal(s string) (Decimal, error) {
	n, ok := normalizeDecimal(s)
	if !ok {
		return "", fmt.Errorf("hyperliquid: invalid decimal %q", s)
	}
	return Decimal(n), nil
}

// DecimalFromFloat returns the shortest decimal representation of f.
// Round prices and sizes with [FormatPrice] and [FormatSize] before placing
// orders.
func DecimalFromFloat(f float64) Decimal {
	return Decimal(strconv.FormatFloat(f, 'f', -1, 64))
}

// String returns d as a string.
func (d Decimal) String() string { return string(d) }

// Float64 parses d as a float64.
func (d Decimal) Float64() (float64, error) { return strconv.ParseFloat(string(d), 64) }

// IsZero reports whether d is empty or numerically zero.
func (d Decimal) IsZero() bool {
	n, ok := normalizeDecimal(string(d))
	return d == "" || ok && n == "0"
}

// MarshalJSON encodes d as a normalized JSON string. An empty Decimal encodes
// as "" and an invalid one is an error.
func (d Decimal) MarshalJSON() ([]byte, error) {
	if d == "" {
		return []byte(`""`), nil
	}
	n, ok := normalizeDecimal(string(d))
	if !ok {
		return nil, fmt.Errorf("hyperliquid: invalid decimal %q", string(d))
	}
	return strconv.AppendQuote(make([]byte, 0, len(n)+2), n), nil
}

// UnmarshalJSON accepts a JSON string or a JSON number.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		s, err := strconv.Unquote(string(b))
		if err != nil {
			return fmt.Errorf("hyperliquid: decode decimal: %w", err)
		}
		*d = Decimal(s)
		return nil
	}
	*d = Decimal(b)
	return nil
}

// normalizeDecimal reports whether s is a plain decimal (optional '-', digits,
// optional fraction; no exponent) and returns its canonical form.
func normalizeDecimal(s string) (string, bool) {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	intPart, frac, _ := strings.Cut(s, ".")
	if intPart == "" && frac == "" || !isDigits(intPart) || !isDigits(frac) {
		return "", false
	}
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	frac = strings.TrimRight(frac, "0")
	n := intPart
	if frac != "" {
		n += "." + frac
	}
	if neg && n != "0" {
		n = "-" + n
	}
	return n, true
}

func isDigits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// ErrTruncatedToZero is returned by [FormatPrice] and [FormatSize] when the
// value has no significant digits left after rounding.
var ErrTruncatedToZero = errors.New("hyperliquid: value truncated to zero")

// FormatPrice rounds px toward zero to a valid Hyperliquid price for an asset
// with the given size decimals: at most 5 significant figures (integers are
// always allowed) and at most 6-szDecimals decimal places for perpetuals or
// 8-szDecimals for spot.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size.
func FormatPrice(px Decimal, szDecimals int, spot bool) (Decimal, error) {
	maxDecimals := 6
	if spot {
		maxDecimals = 8
	}
	n, err := truncateDecimals(px, max(maxDecimals-szDecimals, 0))
	if err != nil {
		return "", err
	}
	intPart, frac, _ := strings.Cut(strings.TrimPrefix(n, "-"), ".")
	if frac != "" {
		// At most 5 significant figures, but never drop integer digits:
		// integer prices are always valid.
		keep := max(5-len(intPart), 0)
		if intPart == "0" {
			keep = len(frac) - len(strings.TrimLeft(frac, "0")) + 5
		}
		n, _ = normalizeDecimal(n[:len(n)-len(frac)+min(len(frac), keep)])
	}
	if n == "0" {
		return "", ErrTruncatedToZero
	}
	return Decimal(n), nil
}

// FormatSize rounds sz toward zero to szDecimals decimal places.
func FormatSize(sz Decimal, szDecimals int) (Decimal, error) {
	n, err := truncateDecimals(sz, szDecimals)
	if err != nil {
		return "", err
	}
	if n == "0" {
		return "", ErrTruncatedToZero
	}
	return Decimal(n), nil
}

// truncateDecimals normalizes d and truncates it to at most places decimals.
func truncateDecimals(d Decimal, places int) (string, error) {
	n, ok := normalizeDecimal(string(d))
	if !ok {
		return "", fmt.Errorf("hyperliquid: invalid decimal %q", string(d))
	}
	if dot := strings.IndexByte(n, '.'); dot >= 0 && len(n)-dot-1 > places {
		n, _ = normalizeDecimal(n[:dot+1+places])
	}
	return n, nil
}
