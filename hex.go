package hyperliquid

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Address is a 20-byte account address. It encodes as a lowercase
// 0x-prefixed hex string, the form Hyperliquid signs and expects.
type Address [20]byte

// ParseAddress parses a 0x-prefixed (or bare) 40-character hex address.
func ParseAddress(s string) (Address, error) {
	var a Address
	return a, decodeHex(a[:], s, "address")
}

// MustParseAddress is like [ParseAddress] but panics on error. It is intended
// for constants and tests.
func MustParseAddress(s string) Address {
	a, err := ParseAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

// String returns the lowercase 0x-prefixed hex form of a.
func (a Address) String() string { return "0x" + hex.EncodeToString(a[:]) }

// IsZero reports whether a is the zero address.
func (a Address) IsZero() bool { return a == Address{} }

// MarshalText implements [encoding.TextMarshaler].
func (a Address) MarshalText() ([]byte, error) { return []byte(a.String()), nil }

// UnmarshalText implements [encoding.TextUnmarshaler].
func (a *Address) UnmarshalText(b []byte) error { return decodeHex(a[:], string(b), "address") }

// Cloid is a 16-byte client order ID, chosen by the client to track an order.
// It encodes as a lowercase 0x-prefixed 32-character hex string.
type Cloid [16]byte

// ParseCloid parses a 0x-prefixed (or bare) 32-character hex client order ID.
func ParseCloid(s string) (Cloid, error) {
	var c Cloid
	return c, decodeHex(c[:], s, "cloid")
}

// MustParseCloid is like [ParseCloid] but panics on error.
func MustParseCloid(s string) Cloid {
	c, err := ParseCloid(s)
	if err != nil {
		panic(err)
	}
	return c
}

// String returns the lowercase 0x-prefixed hex form of c.
func (c Cloid) String() string { return "0x" + hex.EncodeToString(c[:]) }

// MarshalText implements [encoding.TextMarshaler].
func (c Cloid) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

// UnmarshalText implements [encoding.TextUnmarshaler].
func (c *Cloid) UnmarshalText(b []byte) error { return decodeHex(c[:], string(b), "cloid") }

func decodeHex(dst []byte, s, what string) error {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(s) != 2*len(dst) {
		return fmt.Errorf("hyperliquid: invalid %s %q: want %d hex characters", what, s, 2*len(dst))
	}
	if _, err := hex.Decode(dst, []byte(s)); err != nil {
		return fmt.Errorf("hyperliquid: invalid %s %q: %w", what, s, err)
	}
	return nil
}
