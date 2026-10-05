package hyperliquid

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// strictDecoding makes unmarshal reject unknown object fields. The tests
// enable it so that recorded fixtures and the live API checks report fields
// the SDK does not model yet.
var strictDecoding bool

// unmarshal decodes API response JSON into v. Use it, rather than
// json.Unmarshal, wherever a response is decoded into its final type.
func unmarshal(data []byte, v any) error {
	if !strictDecoding {
		return json.Unmarshal(data, v)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("hyperliquid: trailing data after JSON value")
	}
	return nil
}
