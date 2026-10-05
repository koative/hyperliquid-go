package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// InfoClient queries market data and account state through the /info
// endpoint. It needs no keys and is safe for concurrent use.
type InfoClient struct {
	tr       transport
	explorer transport
}

// NewInfoClient returns a client for network.
func NewInfoClient(network Network, opts ...Option) *InfoClient {
	o := newOptions(opts)
	c := &InfoClient{
		tr:       o.transport,
		explorer: &httpTransport{baseURL: network.RPCURL, client: o.httpClient},
	}
	if c.tr == nil {
		c.tr = &httpTransport{baseURL: network.APIURL, client: o.httpClient}
	}
	return c
}

// noParams is the request of info endpoints that take no parameters.
type noParams struct{}

// infoRequest posts {"type": typ, ...req} to /info and decodes the response.
func infoRequest[T any](ctx context.Context, c *InfoClient, typ string, req any) (T, error) {
	return postTyped[T](ctx, c.tr, "info", typ, req)
}

func postTyped[T any](ctx context.Context, tr transport, endpoint, typ string, req any) (T, error) {
	var out T
	body, err := typedJSON(`"type":`+strconv.Quote(typ), req, "")
	if err != nil {
		return out, err
	}
	raw, err := tr.request(ctx, endpoint, body)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("hyperliquid: decode %s response: %w", typ, err)
	}
	return out, nil
}

// unmarshalTuple decodes the JSON array b positionally into dst. Use it to
// implement UnmarshalJSON for Hyperliquid's tuple-shaped values.
func unmarshalTuple(b []byte, dst ...any) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != len(dst) {
		return fmt.Errorf("hyperliquid: want %d-element array, got %d", len(dst), len(raw))
	}
	for i, r := range raw {
		if err := json.Unmarshal(r, dst[i]); err != nil {
			return err
		}
	}
	return nil
}
