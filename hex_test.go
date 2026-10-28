package hyperliquid

import (
	"strings"
	"testing"
)

// A private key passed to ParseAddress by mistake must not end up in logs.
func TestParseHexHidesInput(t *testing.T) {
	key := "0x" + strings.Repeat("ab", 32)
	if _, err := ParseAddress(key); err == nil || strings.Contains(err.Error(), key[2:]) {
		t.Errorf("ParseAddress(key) error = %v; want an error without the input", err)
	}
	if _, err := ParseCloid(strings.Repeat("zz", 16)); err == nil || strings.Contains(err.Error(), "zzzz") {
		t.Errorf("ParseCloid error = %v; want an error without the input", err)
	}
}
