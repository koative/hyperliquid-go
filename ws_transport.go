package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
)

// WithWebSocket sends the requests of an [InfoClient] or [ExchangeClient]
// as WebSocket post requests over ws instead of HTTP. Explorer requests
// still use HTTP. Closing ws fails later requests with
// [ErrWebSocketClosed].
func WithWebSocket(ws *WebSocketClient) Option {
	return func(o *options) { o.transport = ws }
}

// wsResult is the outcome of a post request.
type wsResult struct {
	data json.RawMessage
	err  error
}

// request implements transport by sending a WebSocket post request.
func (c *WebSocketClient) request(ctx context.Context, endpoint string, body []byte) (json.RawMessage, error) {
	typ := "info"
	switch endpoint {
	case "info":
	case "exchange":
		typ = "action"
	default:
		return nil, fmt.Errorf("hyperliquid: %s requests are not supported over websocket", endpoint)
	}
	var (
		conn *websocket.Conn
		id   int64
		ch   = make(chan wsResult, 1)
	)
	for conn == nil {
		c.mu.Lock()
		if c.ctx.Err() != nil {
			c.mu.Unlock()
			return nil, ErrWebSocketClosed
		}
		if conn = c.conn; conn != nil {
			c.lastID++
			id = c.lastID
			c.posts[id] = ch
		}
		up := c.up
		c.mu.Unlock()
		if conn == nil {
			select {
			case <-up:
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-c.ctx.Done():
				return nil, ErrWebSocketClosed
			}
		}
	}
	frame := fmt.Appendf(nil, `{"method":"post","id":%d,"request":{"type":%q,"payload":%s}}`, id, typ, body)
	if err := writeFrame(conn, frame); err != nil {
		c.dropPost(id)
		if c.ctx.Err() != nil {
			return nil, ErrWebSocketClosed
		}
		// The frame may have partly reached the server, like any request
		// in flight when the connection drops.
		return nil, fmt.Errorf("%w: %w", ErrWebSocketDisconnected, err)
	}
	select {
	case r := <-ch:
		return r.data, r.err
	case <-ctx.Done():
		c.dropPost(id)
		return nil, ctx.Err()
	}
}

func (c *WebSocketClient) dropPost(id int64) {
	c.mu.Lock()
	delete(c.posts, id)
	c.mu.Unlock()
}

func (c *WebSocketClient) resolvePost(id int64, r wsResult) bool {
	c.mu.Lock()
	ch, ok := c.posts[id]
	delete(c.posts, id)
	c.mu.Unlock()
	if ok {
		ch <- r
	}
	return ok
}

func (c *WebSocketClient) handlePost(data json.RawMessage) {
	var p struct {
		ID       int64 `json:"id"`
		Response struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		c.report(fmt.Errorf("hyperliquid: decode websocket post response: %w", err))
		return
	}
	var r wsResult
	switch p.Response.Type {
	case "info":
		var info struct {
			Data json.RawMessage `json:"data"`
		}
		r.err = json.Unmarshal(p.Response.Payload, &info)
		r.data = info.Data
	case "action":
		r.data = p.Response.Payload
	default: // "error"
		var msg string
		if json.Unmarshal(p.Response.Payload, &msg) != nil {
			msg = string(p.Response.Payload)
		}
		r.err = &APIError{Message: msg}
	}
	c.resolvePost(p.ID, r)
}
