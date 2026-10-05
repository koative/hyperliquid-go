// Package msgpack re-encodes JSON as MessagePack exactly the way Python's
// msgpack.packb encodes the equivalent dict: object keys keep their order,
// integers use the smallest encoding and strings of 32-255 bytes use str8.
//
// Hyperliquid hashes msgpack(action) when signing L1 actions. Deriving the
// msgpack bytes from the JSON request body guarantees that what is signed is
// exactly what is sent.
package msgpack

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FromJSON converts the single JSON value in data to MessagePack. Numbers
// must be integers: Hyperliquid actions carry decimals as strings, so a
// fractional number indicates a bug.
func FromJSON(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	out, err := appendValue(nil, dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("msgpack: trailing data after JSON value")
	}
	return out, nil
}

func appendValue(out []byte, dec *json.Decoder) ([]byte, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("msgpack: %w", err)
	}
	switch t := tok.(type) {
	case json.Delim:
		var body []byte
		n := 0
		for dec.More() {
			if t == '{' {
				key, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("msgpack: %w", err)
				}
				body = appendString(body, key.(string))
			}
			if body, err = appendValue(body, dec); err != nil {
				return nil, err
			}
			n++
		}
		if _, err := dec.Token(); err != nil { // closing delimiter
			return nil, fmt.Errorf("msgpack: %w", err)
		}
		if t == '{' {
			out = appendHeader(out, n, 0x80, 0xde, 0xdf)
		} else {
			out = appendHeader(out, n, 0x90, 0xdc, 0xdd)
		}
		return append(out, body...), nil
	case string:
		return appendString(out, t), nil
	case json.Number:
		return appendInt(out, t.String())
	case bool:
		if t {
			return append(out, 0xc3), nil
		}
		return append(out, 0xc2), nil
	case nil:
		return append(out, 0xc0), nil
	}
	return nil, fmt.Errorf("msgpack: unexpected JSON token %v", tok)
}

// appendHeader writes a map or array header: fix form for n < 16, else the
// 16- or 32-bit form.
func appendHeader(out []byte, n int, fix, b16, b32 byte) []byte {
	switch {
	case n < 16:
		return append(out, fix|byte(n))
	case n <= 0xffff:
		return binary.BigEndian.AppendUint16(append(out, b16), uint16(n))
	default:
		return binary.BigEndian.AppendUint32(append(out, b32), uint32(n))
	}
}

func appendString(out []byte, s string) []byte {
	switch n := len(s); {
	case n < 32:
		out = append(out, 0xa0|byte(n))
	case n <= 0xff:
		out = append(out, 0xd9, byte(n))
	case n <= 0xffff:
		out = binary.BigEndian.AppendUint16(append(out, 0xda), uint16(n))
	default:
		out = binary.BigEndian.AppendUint32(append(out, 0xdb), uint32(n))
	}
	return append(out, s...)
}

func appendInt(out []byte, s string) ([]byte, error) {
	if strings.ContainsAny(s, ".eE") {
		return nil, fmt.Errorf("msgpack: non-integer number %s (decimals must be strings)", s)
	}
	if !strings.HasPrefix(s, "-") {
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("msgpack: %w", err)
		}
		switch {
		case u < 0x80:
			return append(out, byte(u)), nil
		case u <= 0xff:
			return append(out, 0xcc, byte(u)), nil
		case u <= 0xffff:
			return binary.BigEndian.AppendUint16(append(out, 0xcd), uint16(u)), nil
		case u <= 0xffffffff:
			return binary.BigEndian.AppendUint32(append(out, 0xce), uint32(u)), nil
		default:
			return binary.BigEndian.AppendUint64(append(out, 0xcf), u), nil
		}
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("msgpack: %w", err)
	}
	switch {
	case i >= -32:
		return append(out, byte(i)), nil
	case i >= -128:
		return append(out, 0xd0, byte(i)), nil
	case i >= -32768:
		return binary.BigEndian.AppendUint16(append(out, 0xd1), uint16(i)), nil
	case i >= -1<<31:
		return binary.BigEndian.AppendUint32(append(out, 0xd2), uint32(i)), nil
	default:
		return binary.BigEndian.AppendUint64(append(out, 0xd3), uint64(i)), nil
	}
}
