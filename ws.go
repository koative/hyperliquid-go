package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

var (
	// ErrWebSocketClosed is returned by [WebSocketClient] methods after
	// [WebSocketClient.Close], and by [Subscription.Err] for subscriptions
	// ended by it.
	ErrWebSocketClosed = errors.New("hyperliquid: websocket closed")
	// ErrWebSocketDisconnected is returned by requests in flight when the
	// connection drops. The request may or may not have been executed.
	ErrWebSocketDisconnected = errors.New("hyperliquid: websocket disconnected")
	// ErrSubscriptionOverflow is returned by [Subscription.Err] for a
	// subscription ended because its handler fell more than 4096 messages
	// behind.
	ErrSubscriptionOverflow = errors.New("hyperliquid: websocket subscription handler fell behind")
)

// Server-side limits, enforced client-side because the server rejects the
// excess without saying which request it refers to. The rate-limits page
// (https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/rate-limits-and-user-limits)
// documents 10 users, but mainnet accepts 15 per connection and rejects the
// 16th with "Cannot track more than 15 total users." (measured 2026-10-05).
const (
	maxWebSocketSubscriptions = 1000
	maxWebSocketUsers         = 15
)

// maxSubscriptionQueue is the number of messages a [Subscription] buffers
// for a handler that falls behind.
const maxSubscriptionQueue = 4096

// WebSocketClient is a connection to the Hyperliquid WebSocket API. It
// streams subscriptions and, through [WithWebSocket], carries
// [InfoClient] and [ExchangeClient] requests.
//
// The client pings the server every 30 seconds and reconnects with capped
// exponential backoff when the connection drops or a ping goes
// unanswered, then re-subscribes every live subscription. Requests in
// flight when the connection drops fail with [ErrWebSocketDisconnected];
// requests made while reconnecting wait for the new connection.
//
// Subscription methods return once the server confirms the subscription;
// their ctx bounds only that handshake, not the subscription's lifetime.
// Subscriptions with an identical request share one server subscription.
// A subscription that joins a live one does not receive the stream's
// initial snapshot, such as the first full frame of
// [WebSocketClient.FastAssetCtxs] or the history of
// [WebSocketClient.UserFills]; it receives only later messages. See
// https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/websocket/subscriptions
// for the streams.
//
// A WebSocketClient is safe for concurrent use. Call
// [WebSocketClient.Close] to release it.
type WebSocketClient struct {
	url  string
	cfg  wsConfig
	ctx  context.Context // canceled by Close
	stop context.CancelFunc
	done chan struct{} // closed when run returns

	mu   sync.Mutex
	conn *websocket.Conn // nil while disconnected
	up   chan struct{}   // closed while conn is set
	// resubscribed is closed once the re-subscribe frames have been written
	// to conn; subscribe and unsubscribe frames wait for it so they cannot
	// overtake the burst.
	resubscribed chan struct{}
	lastID       int64
	posts        map[int64]chan wsResult
	pending      []*wsRequest      // subscribe/unsubscribe awaiting the server's echo
	subs         map[string]*wsSub // by wire subscription payload
}

// WebSocketOption configures a [WebSocketClient].
type WebSocketOption func(*wsConfig)

type wsConfig struct {
	onError                   func(error)
	pingInterval, pongTimeout time.Duration
	minBackoff, maxBackoff    time.Duration
}

// WithWebSocketErrorHandler sets a function that receives errors that
// belong to no request: lost connections (wrapping
// [ErrWebSocketDisconnected]), reconnection failures, server errors that
// match no request, and messages that fail to decode. It runs on the
// client's internal goroutines, possibly concurrently, so it must return
// promptly and must not call [WebSocketClient.Close], subscribe, or send
// requests through the client. By default these errors are discarded.
func WithWebSocketErrorHandler(f func(error)) WebSocketOption {
	return func(c *wsConfig) { c.onError = f }
}

// DialWebSocket connects to the WebSocket API of network.
func DialWebSocket(ctx context.Context, network Network, opts ...WebSocketOption) (*WebSocketClient, error) {
	return dialWebSocket(ctx, network.WebSocketURL, opts)
}

// DialExplorerWebSocket connects to the WebSocket of network's explorer RPC,
// which serves only [WebSocketClient.ExplorerBlock] and
// [WebSocketClient.ExplorerTxs].
func DialExplorerWebSocket(ctx context.Context, network Network, opts ...WebSocketOption) (*WebSocketClient, error) {
	return dialWebSocket(ctx, "ws"+strings.TrimPrefix(network.RPCURL, "http")+"/ws", opts)
}

func dialWebSocket(ctx context.Context, url string, opts []WebSocketOption) (*WebSocketClient, error) {
	c := &WebSocketClient{
		url: url,
		cfg: wsConfig{
			pingInterval: 30 * time.Second,
			pongTimeout:  10 * time.Second,
			minBackoff:   500 * time.Millisecond,
			maxBackoff:   30 * time.Second,
		},
		done:  make(chan struct{}),
		up:    make(chan struct{}),
		posts: make(map[int64]chan wsResult),
		subs:  make(map[string]*wsSub),
	}
	for _, o := range opts {
		o(&c.cfg)
	}
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	c.ctx, c.stop = context.WithCancel(context.Background())
	go c.run(conn)
	return c, nil
}

func (c *WebSocketClient) connect(ctx context.Context) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, c.url, nil) //nolint:bodyclose // coder/websocket closes the handshake response body
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: dial %s: %w", c.url, err)
	}
	conn.SetReadLimit(-1) // snapshots such as webData2 run to megabytes
	return conn, nil
}

// Close closes the connection and ends every subscription with
// [ErrWebSocketClosed]. It returns once the client's goroutines, except
// subscription handlers still running, have exited.
func (c *WebSocketClient) Close() error {
	c.stop()
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}
	<-c.done
	return nil
}

func (c *WebSocketClient) report(err error) {
	if c.cfg.onError != nil {
		c.cfg.onError(err)
	}
}

// run serves conn, then reconnects until Close.
func (c *WebSocketClient) run(conn *websocket.Conn) {
	defer close(c.done)
	defer c.shutdown()
	backoff := c.cfg.minBackoff
	for {
		start := time.Now()
		c.serve(conn)
		if time.Since(start) > c.cfg.maxBackoff {
			backoff = c.cfg.minBackoff
		}
		for conn = nil; conn == nil; {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(backoff/2 + rand.N(backoff/2+1)): //nolint:gosec // jitter needs no cryptographic randomness
			}
			backoff = min(2*backoff, c.cfg.maxBackoff)
			var err error
			if conn, err = c.connect(c.ctx); err != nil && c.ctx.Err() == nil {
				c.report(err)
			}
		}
	}
}

// shutdown ends every subscription once the client is closed.
func (c *WebSocketClient) shutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.subs {
		c.failSub(s, ErrWebSocketClosed)
	}
}

// serve re-subscribes, then reads conn until it fails.
func (c *WebSocketClient) serve(conn *websocket.Conn) {
	c.mu.Lock()
	if c.ctx.Err() != nil { // Close ran before conn was published
		c.mu.Unlock()
		_ = conn.CloseNow()
		return
	}
	c.conn = conn
	close(c.up)
	resubscribed := make(chan struct{})
	c.resubscribed = resubscribed
	frames := make([][]byte, 0, len(c.subs))
	for _, s := range c.subs {
		c.pending = append(c.pending, &wsRequest{method: "subscribe", payload: decodeAny(s.payload), sub: s})
		frames = append(frames, wsFrame("subscribe", s.payload))
	}
	c.mu.Unlock()
	for _, f := range frames {
		if writeFrame(conn, f) != nil {
			break
		}
	}
	close(resubscribed)

	stop := make(chan struct{})
	pongs := make(chan struct{}, 1)
	var wg sync.WaitGroup
	wg.Go(func() { c.keepAlive(conn, stop, pongs) })
	err := c.read(conn, pongs)
	close(stop)
	wg.Wait()
	_ = conn.CloseNow()

	closed := c.ctx.Err() != nil
	if !closed {
		c.report(fmt.Errorf("%w: %w", ErrWebSocketDisconnected, err))
	}
	lost := ErrWebSocketDisconnected
	if closed {
		lost = ErrWebSocketClosed
	}
	c.mu.Lock()
	c.conn = nil
	c.up = make(chan struct{})
	for id, ch := range c.posts {
		ch <- wsResult{err: lost}
		delete(c.posts, id)
	}
	// The server forgets every subscription with the connection; subscribes
	// are re-sent on the next one.
	for _, r := range c.pending {
		if r.done != nil {
			r.done <- nil
		}
	}
	c.pending = nil
	c.mu.Unlock()
}

func (c *WebSocketClient) keepAlive(conn *websocket.Conn, stop <-chan struct{}, pongs chan struct{}) {
	t := time.NewTicker(c.cfg.pingInterval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
		}
		select {
		case <-pongs: // stale
		default:
		}
		if writeFrame(conn, []byte(`{"method":"ping"}`)) != nil {
			return
		}
		select {
		case <-stop:
			return
		case <-pongs:
		case <-time.After(c.cfg.pongTimeout):
			_ = conn.CloseNow() // half-open connection: make read fail
			return
		}
	}
}

// writeFrame writes with its own timeout: coder/websocket closes the
// connection when a write's context ends, so a caller's context must never
// reach it.
func writeFrame(conn *websocket.Conn, frame []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, frame)
}

func wsFrame(method string, subscription []byte) []byte {
	return fmt.Appendf(nil, `{"method":%q,"subscription":%s}`, method, subscription)
}

func (c *WebSocketClient) read(conn *websocket.Conn, pongs chan struct{}) error {
	for {
		_, b, err := conn.Read(context.Background())
		if err != nil {
			return err
		}
		switch b = bytes.TrimSpace(b); {
		case len(b) == 0: // the greeting "Websocket connection established." and other non-JSON text
		case b[0] == '[': // explorer pushes arrays without an envelope
			var p wsProbe
			if json.Unmarshal(b, &p) == nil {
				switch {
				case p.BlockTime != nil:
					c.dispatch("explorerBlock", b, &p)
				case p.Action != nil:
					c.dispatch("explorerTxs", b, &p)
				}
			}
		case b[0] == '{':
			var f struct {
				Channel string          `json:"channel"`
				Data    json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(b, &f); err != nil {
				c.report(fmt.Errorf("hyperliquid: decode websocket message: %w", err))
				continue
			}
			switch f.Channel {
			case "pong":
				select {
				case pongs <- struct{}{}:
				default:
				}
			case "post":
				c.handlePost(f.Data)
			case "subscriptionResponse":
				var echo struct {
					Method       string          `json:"method"`
					Subscription json.RawMessage `json:"subscription"`
				}
				if json.Unmarshal(f.Data, &echo) == nil {
					c.resolveEcho(echo.Method, decodeAny(echo.Subscription), nil)
				}
			case "error":
				c.handleError(f.Data)
			default:
				c.dispatch(f.Channel, f.Data, nil)
			}
		}
	}
}

// dispatch queues data for every listener of channel whose filter accepts
// it. probe, if nil, is decoded from data on first use.
func (c *WebSocketClient) dispatch(channel string, data json.RawMessage, probe *wsProbe) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// ponytail: linear scan of at most 1000 subscriptions per message; index by channel if it shows in profiles.
	for _, s := range c.subs {
		if s.channel != channel {
			continue
		}
		if s.match != nil {
			if probe == nil {
				probe = new(wsProbe)
				_ = json.Unmarshal(data, probe) // a mismatched field leaves the probe partly empty, failing filters
			}
			if !s.match(probe) {
				continue
			}
		}
		for l := range s.listeners {
			if !l.push(data) {
				if f, _ := c.detachLocked(l); f.conn != nil {
					go f.write() // the reader must not block
				}
			}
		}
	}
}

// wsProbe holds the event fields that route a message to subscriptions.
type wsProbe struct {
	Coin      string `json:"coin"`
	User      string `json:"user"`
	Dex       string `json:"dex"`
	S         string `json:"s"` // candle coin
	I         string `json:"i"` // candle interval
	UserState struct {
		User string `json:"user"`
	} `json:"userState"`
	BlockTime json.RawMessage `json:"blockTime"`
	Action    json.RawMessage `json:"action"`
}

// UnmarshalJSON decodes an object, or the first element of an array (the
// trades and explorer events).
func (p *wsProbe) UnmarshalJSON(b []byte) error {
	type plain wsProbe
	if len(b) == 0 || b[0] != '[' {
		return json.Unmarshal(b, (*plain)(p))
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil || !dec.More() {
		return err
	}
	return dec.Decode((*plain)(p))
}

var trailingID = regexp.MustCompile(`id=(\d+)$`)

// handleError routes an "error" message to the request it names; see
// https://github.com/nktkas/hyperliquid src/transport/websocket/_dispatcher.ts.
func (c *WebSocketClient) handleError(data json.RawMessage) {
	var msg string
	if json.Unmarshal(data, &msg) != nil {
		msg = string(data)
	}
	err := &APIError{Message: msg}
	if m := trailingID.FindStringSubmatch(msg); m != nil {
		if id, perr := strconv.ParseInt(m[1], 10, 64); perr == nil && c.resolvePost(id, wsResult{err: err}) {
			return
		}
	}
	if i, j := strings.IndexByte(msg, '{'), strings.LastIndexByte(msg, '}'); i >= 0 && j > i {
		if echo, ok := decodeAny([]byte(msg[i : j+1])).(map[string]any); ok {
			id, isPost := echo["id"].(float64)
			sub, isSub := echo["subscription"]
			method, _ := echo["method"].(string)
			switch {
			case isPost:
				if c.resolvePost(int64(id), wsResult{err: err}) {
					return
				}
			case isSub:
				if c.resolveEcho(method, sub, err) {
					return
				}
			case strings.HasPrefix(msg, "Already subscribed"), strings.HasPrefix(msg, "Invalid subscription"):
				if c.resolveEcho("subscribe", echo, err) {
					return
				}
			case strings.HasPrefix(msg, "Already unsubscribed"):
				if c.resolveEcho("unsubscribe", echo, err) {
					return
				}
			}
		}
	}
	c.report(err)
}

// decodeAny decodes JSON into maps, slices and scalars, or returns nil.
func decodeAny(b []byte) any {
	var v any
	_ = json.Unmarshal(b, &v)
	return v
}

// wsSub is a server-side subscription shared by every [Subscription] with
// the same payload.
type wsSub struct {
	payload   []byte // wire JSON
	channel   string
	user      string // lowercase, for the user limit
	match     func(*wsProbe) bool
	listeners map[*Subscription]struct{}
	ready     chan struct{} // closed once the first subscribe is answered
	settled   bool
	err       error // why the first subscribe failed
}

// settle records the answer to the first subscribe. c.mu must be held.
func (s *wsSub) settle(err error) {
	if !s.settled {
		s.settled, s.err = true, err
		close(s.ready)
	}
}

// wsRequest is a subscribe or unsubscribe awaiting the server's echo.
type wsRequest struct {
	method  string
	payload any        // decoded, for matching the echo
	sub     *wsSub     // subscribe
	done    chan error // unsubscribe
}

// Subscription is a stream created by a [WebSocketClient] method. Its
// handler receives messages one at a time, in order, on a goroutine of its
// own; up to 4096 messages wait for a handler that falls behind. A
// Subscription ends when it is unsubscribed, when the client is closed,
// when its handler falls further behind, or when the server rejects it
// after a reconnect.
type Subscription struct {
	c       *WebSocketClient
	sub     *wsSub
	deliver func(json.RawMessage)
	wake    chan struct{}
	done    chan struct{}

	mu    sync.Mutex
	queue []json.RawMessage
	ended bool
	err   error
}

// Done returns a channel that is closed when the subscription ends.
func (l *Subscription) Done() <-chan struct{} { return l.done }

// Err returns why the subscription ended: nil while it is live or after
// [Subscription.Unsubscribe], [ErrWebSocketClosed] after
// [WebSocketClient.Close], [ErrSubscriptionOverflow] if its handler fell
// behind, or the server's [*APIError].
func (l *Subscription) Err() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.err
}

// Unsubscribe ends the subscription. The server subscription is removed,
// and Unsubscribe waits for the server to confirm, once no other
// [Subscription] shares it. The handler is not called after Unsubscribe
// returns, except for a call already in progress.
func (l *Subscription) Unsubscribe(ctx context.Context) error {
	f, r := l.c.detach(l)
	l.finish(nil)
	if r == nil {
		return nil
	}
	f.write()
	select {
	case err := <-r.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// push queues b for the handler. It reports false if the queue was full,
// in which case it ends l with [ErrSubscriptionOverflow].
func (l *Subscription) push(b json.RawMessage) bool {
	l.mu.Lock()
	switch {
	case l.ended:
		l.mu.Unlock()
		return true
	case len(l.queue) >= maxSubscriptionQueue:
		l.mu.Unlock()
		l.finish(ErrSubscriptionOverflow)
		return false
	}
	l.queue = append(l.queue, b)
	l.mu.Unlock()
	select {
	case l.wake <- struct{}{}:
	default:
	}
	return true
}

func (l *Subscription) loop() {
	for {
		select {
		case <-l.done:
			return
		case <-l.wake:
		}
		for {
			l.mu.Lock()
			if l.ended || len(l.queue) == 0 {
				l.mu.Unlock()
				break
			}
			b := l.queue[0]
			l.queue[0] = nil
			l.queue = l.queue[1:]
			l.mu.Unlock()
			l.deliver(b)
		}
	}
}

func (l *Subscription) finish(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.ended {
		l.ended, l.err, l.queue = true, err, nil
		close(l.done)
	}
}

// subscribe sends the subscription {"type": typ, ...req} and passes the
// messages of channel accepted by match (all if nil) to handler.
func subscribe[T any](ctx context.Context, c *WebSocketClient, typ, channel string, req any, match func(*wsProbe) bool, handler func(T)) (*Subscription, error) {
	wire, err := typedJSON(`"type":`+strconv.Quote(typ), req, "")
	if err != nil {
		return nil, err
	}
	l := &Subscription{c: c, wake: make(chan struct{}, 1), done: make(chan struct{})}
	l.deliver = func(b json.RawMessage) {
		var v T
		if err := unmarshal(b, &v); err != nil {
			c.report(fmt.Errorf("hyperliquid: decode %s message: %w", channel, err))
			return
		}
		handler(v)
	}

	c.mu.Lock()
	if c.ctx.Err() != nil {
		c.mu.Unlock()
		return nil, ErrWebSocketClosed
	}
	s := c.subs[string(wire)]
	var f subFrame
	if s == nil {
		var p struct {
			User string `json:"user"`
		}
		_ = json.Unmarshal(wire, &p)
		if err := c.checkLimits(strings.ToLower(p.User)); err != nil {
			c.mu.Unlock()
			return nil, err
		}
		s = &wsSub{
			payload:   wire,
			channel:   channel,
			user:      strings.ToLower(p.User),
			match:     match,
			listeners: make(map[*Subscription]struct{}),
			ready:     make(chan struct{}),
		}
		c.subs[string(wire)] = s
		if c.conn != nil {
			c.pending = append(c.pending, &wsRequest{method: "subscribe", payload: decodeAny(wire), sub: s})
			f = subFrame{c.conn, c.resubscribed, wsFrame("subscribe", wire)}
		}
	}
	s.listeners[l] = struct{}{}
	l.sub = s
	c.mu.Unlock()

	go l.loop()
	f.write()
	select {
	case <-s.ready:
		if s.err != nil {
			return nil, s.err
		}
		return l, nil
	case <-ctx.Done():
		f, _ := c.detach(l)
		f.write()
		l.finish(nil)
		return nil, ctx.Err()
	}
}

// subFrame is a subscribe or unsubscribe frame for conn, written once the
// re-subscribe burst of conn is out so that it cannot overtake it.
type subFrame struct {
	conn  *websocket.Conn // nil: nothing to write
	after <-chan struct{}
	frame []byte
}

func (f subFrame) write() {
	if f.conn == nil {
		return
	}
	<-f.after
	// On failure the connection drops: subscribes are re-sent on the next
	// one and pending unsubscribes are answered.
	_ = writeFrame(f.conn, f.frame)
}

func (c *WebSocketClient) checkLimits(user string) error {
	if len(c.subs) >= maxWebSocketSubscriptions {
		return fmt.Errorf("hyperliquid: cannot hold more than %d websocket subscriptions", maxWebSocketSubscriptions)
	}
	if user == "" {
		return nil
	}
	users := map[string]bool{}
	for _, s := range c.subs {
		if s.user != "" {
			users[s.user] = true
		}
	}
	if !users[user] && len(users) >= maxWebSocketUsers {
		return fmt.Errorf("hyperliquid: cannot track more than %d websocket users", maxWebSocketUsers)
	}
	return nil
}

// detach removes l from its server subscription. When l was the last
// listener and the client is connected, it returns the unsubscribe frame
// to write and the request awaiting its answer.
func (c *WebSocketClient) detach(l *Subscription) (subFrame, *wsRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.detachLocked(l)
}

// detachLocked is [WebSocketClient.detach] with c.mu held.
func (c *WebSocketClient) detachLocked(l *Subscription) (subFrame, *wsRequest) {
	s := l.sub
	if _, ok := s.listeners[l]; !ok {
		return subFrame{}, nil
	}
	delete(s.listeners, l)
	if len(s.listeners) > 0 || c.subs[string(s.payload)] != s {
		return subFrame{}, nil
	}
	delete(c.subs, string(s.payload))
	if c.conn == nil {
		return subFrame{}, nil
	}
	r := &wsRequest{method: "unsubscribe", payload: decodeAny(s.payload), done: make(chan error, 1)}
	c.pending = append(c.pending, r)
	return subFrame{c.conn, c.resubscribed, wsFrame("unsubscribe", s.payload)}, r
}

// failSub ends s and its listeners with err. c.mu must be held.
func (c *WebSocketClient) failSub(s *wsSub, err error) {
	if c.subs[string(s.payload)] == s {
		delete(c.subs, string(s.payload))
	}
	s.settle(err)
	for l := range s.listeners {
		l.finish(err)
	}
	clear(s.listeners)
}

// resolveEcho answers the pending request whose payload the server echoed
// with err. The server adds defaults to its echo, so the most specific
// pending payload that is a subset of the echo wins.
func (c *WebSocketClient) resolveEcho(method string, echo any, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	best, bestN := -1, -1
	for i, r := range c.pending {
		if r.method == method && jsonSubset(r.payload, echo) {
			if n := jsonLeaves(r.payload); n > bestN {
				best, bestN = i, n
			}
		}
	}
	if best < 0 {
		return false
	}
	r := c.pending[best]
	c.pending = slices.Delete(c.pending, best, best+1)
	switch {
	case r.sub == nil:
		r.done <- err
	case err != nil:
		c.failSub(r.sub, err)
	default:
		r.sub.settle(nil)
	}
	return true
}

// jsonSubset reports whether every value in sub is present in sup,
// comparing hex strings case-insensitively.
func jsonSubset(sub, sup any) bool {
	switch a := sub.(type) {
	case map[string]any:
		b, ok := sup.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range a {
			if w, ok := b[k]; !ok || !jsonSubset(v, w) {
				return false
			}
		}
		return true
	case []any:
		b, ok := sup.([]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !jsonSubset(a[i], b[i]) {
				return false
			}
		}
		return true
	case string:
		b, ok := sup.(string)
		return ok && (a == b || strings.HasPrefix(a, "0x") && strings.EqualFold(a, b))
	default:
		return sub == sup
	}
}

// jsonLeaves counts the scalar values in v.
func jsonLeaves(v any) int {
	n := 0
	switch v := v.(type) {
	case map[string]any:
		for _, x := range v {
			n += jsonLeaves(x)
		}
	case []any:
		for _, x := range v {
			n += jsonLeaves(x)
		}
	default:
		n = 1
	}
	return n
}
