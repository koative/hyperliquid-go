package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Network identifies a Hyperliquid deployment.
type Network struct {
	// APIURL is the HTTP API base URL, e.g. "https://api.hyperliquid.xyz".
	APIURL string
	// WebSocketURL is the WebSocket API URL, e.g. "wss://api.hyperliquid.xyz/ws".
	WebSocketURL string
	// RPCURL is the base URL of the explorer RPC, e.g. "https://rpc.hyperliquid.xyz".
	RPCURL string
	// Mainnet selects mainnet signatures. Signatures made for one network
	// are rejected by the other.
	Mainnet bool
}

var (
	// Mainnet is the Hyperliquid mainnet.
	Mainnet = Network{
		APIURL:       "https://api.hyperliquid.xyz",
		WebSocketURL: "wss://api.hyperliquid.xyz/ws",
		RPCURL:       "https://rpc.hyperliquid.xyz",
		Mainnet:      true,
	}
	// Testnet is the Hyperliquid testnet.
	Testnet = Network{
		APIURL:       "https://api.hyperliquid-testnet.xyz",
		WebSocketURL: "wss://api.hyperliquid-testnet.xyz/ws",
		RPCURL:       "https://rpc.hyperliquid-testnet.xyz",
	}
)

// Option configures an [InfoClient] or [ExchangeClient].
type Option func(*options)

type options struct {
	httpClient *http.Client
	transport  transport
}

func newOptions(opts []Option) options {
	o := options{httpClient: &http.Client{Timeout: 10 * time.Second}}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithHTTPClient sets the HTTP client used for requests. The default client
// has a 10 second timeout.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// transport sends a JSON request to the /info or /exchange endpoint and
// returns the raw JSON response.
type transport interface {
	request(ctx context.Context, endpoint string, body []byte) (json.RawMessage, error)
}

type httpTransport struct {
	baseURL string
	client  *http.Client
}

func (t *httpTransport) request(ctx context.Context, endpoint string, body []byte) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/"+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: read %s response: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK || !json.Valid(b) {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(bytes.TrimSpace(b))}
	}
	return b, nil
}

// APIError is an error reported by the Hyperliquid API.
type APIError struct {
	// StatusCode is the HTTP status code. It is 0 only for errors reported
	// inside an ok /exchange response body or over WebSocket.
	StatusCode int
	// Message is the server's error message.
	Message string
}

func (e *APIError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("hyperliquid: %s (HTTP %d)", e.Message, e.StatusCode)
	}
	return "hyperliquid: " + e.Message
}
