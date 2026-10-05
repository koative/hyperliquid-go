package hyperliquid

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// wsServer is an in-process stand-in for the Hyperliquid WebSocket API. It
// echoes subscriptions with an added field, as the real server adds
// defaults, rejects subscriptions whose coin is "BAD", and answers posts.
type wsServer struct {
	*httptest.Server
	connected chan struct{}
	mutePings atomic.Bool

	mu   sync.Mutex
	conn *websocket.Conn // latest
	subs []string        // "subscribe {...}" frames received, in order
}

func newWSServer(t *testing.T) *wsServer {
	s := &wsServer{connected: make(chan struct{}, 10)}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.CloseNow() }()
		ctx := context.Background()
		if c.Write(ctx, websocket.MessageText, []byte("Websocket connection established.")) != nil {
			return
		}
		s.mu.Lock()
		s.conn = c
		s.mu.Unlock()
		s.connected <- struct{}{}
		for {
			_, b, err := c.Read(ctx)
			if err != nil {
				return
			}
			var m struct {
				Method       string         `json:"method"`
				ID           int64          `json:"id"`
				Subscription map[string]any `json:"subscription"`
				Request      struct {
					Type    string         `json:"type"`
					Payload map[string]any `json:"payload"`
				} `json:"request"`
			}
			if err := json.Unmarshal(b, &m); err != nil {
				t.Errorf("server: bad frame %s", b)
				return
			}
			var reply any
			if m.Method == "ping" && s.mutePings.Load() {
				continue
			}
			switch m.Method {
			case "ping":
				reply = map[string]any{"channel": "pong"}
			case "subscribe", "unsubscribe":
				sub, _ := json.Marshal(m.Subscription)
				s.mu.Lock()
				s.subs = append(s.subs, m.Method+" "+string(sub))
				s.mu.Unlock()
				if m.Subscription["coin"] == "BAD" {
					reply = map[string]any{"channel": "error", "data": "Invalid subscription " + string(sub)}
					break
				}
				m.Subscription["serverDefault"] = nil
				reply = map[string]any{"channel": "subscriptionResponse", "data": map[string]any{"method": m.Method, "subscription": m.Subscription}}
			case "post": //nolint:usestdlibvars // a WebSocket method, not HTTP
				var resp map[string]any
				switch {
				case m.Request.Type == "action":
					resp = map[string]any{"type": "action", "payload": map[string]any{"status": "ok", "response": map[string]any{"type": "default"}}}
				case m.Request.Payload["type"] == "allMids":
					resp = map[string]any{"type": "info", "payload": map[string]any{"type": "allMids", "data": map[string]any{"BTC": "100.5"}}}
				default:
					resp = map[string]any{"type": "error", "payload": "unknown request"}
				}
				reply = map[string]any{"channel": "post", "data": map[string]any{"id": m.ID, "response": resp}}
			}
			out, _ := json.Marshal(reply)
			if c.Write(ctx, websocket.MessageText, out) != nil {
				return
			}
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *wsServer) network() Network {
	return Network{WebSocketURL: "ws" + strings.TrimPrefix(s.URL, "http")}
}

func (s *wsServer) push(t *testing.T, frame string) {
	t.Helper()
	s.mu.Lock()
	c := s.conn
	s.mu.Unlock()
	if err := c.Write(context.Background(), websocket.MessageText, []byte(frame)); err != nil {
		t.Fatal(err)
	}
}

func (s *wsServer) received() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.subs...)
}

func dialTest(t *testing.T, s *wsServer, opts ...WebSocketOption) *WebSocketClient {
	t.Helper()
	opts = append(opts, func(c *wsConfig) {
		c.minBackoff, c.maxBackoff = time.Millisecond, 10*time.Millisecond
	})
	ws, err := DialWebSocket(t.Context(), s.network(), opts...)
	if err != nil {
		t.Fatal(err)
	}
	<-s.connected
	t.Cleanup(func() { _ = ws.Close() })
	return ws
}

// collect returns a handler that sends what it receives to the returned
// channel.
func collect[T any]() (func(T), chan T) {
	ch := make(chan T, 10)
	return func(v T) { ch <- v }, ch
}

func next[T any](t *testing.T, ch chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a message")
		panic("unreachable")
	}
}

func none[T any](t *testing.T, ch chan T) {
	t.Helper()
	select {
	case v := <-ch:
		t.Fatalf("unexpected message %+v", v)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestWebSocketDispatch(t *testing.T) {
	s := newWSServer(t)
	ws := dialTest(t, s)
	ctx := t.Context()

	btcA, btcACh := collect[[]Trade]()
	btcB, btcBCh := collect[[]Trade]()
	eth, ethCh := collect[[]Trade]()
	candle, candleCh := collect[Candle]()
	user := MustParseAddress("0x00000000000000000000000000000000000000AA")
	ch, chCh := collect[ClearinghouseStateEvent]()
	for _, sub := range []func() (*Subscription, error){
		func() (*Subscription, error) { return ws.Trades(ctx, TradesSubscription{Coin: "BTC"}, btcA) },
		func() (*Subscription, error) { return ws.Trades(ctx, TradesSubscription{Coin: "BTC"}, btcB) },
		func() (*Subscription, error) { return ws.Trades(ctx, TradesSubscription{Coin: "ETH"}, eth) },
		func() (*Subscription, error) {
			return ws.Candle(ctx, CandleSubscription{Coin: "BTC", Interval: Candle1m}, candle)
		},
		func() (*Subscription, error) {
			return ws.ClearinghouseState(ctx, ClearinghouseStateSubscription{User: user}, ch)
		},
	} {
		if _, err := sub(); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		`subscribe {"coin":"BTC","type":"trades"}`,
		`subscribe {"coin":"ETH","type":"trades"}`,
		`subscribe {"coin":"BTC","interval":"1m","type":"candle"}`,
		`subscribe {"dex":"","type":"clearinghouseState","user":"0x00000000000000000000000000000000000000aa"}`,
	}
	if got := s.received(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("server subscriptions:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	s.push(t, `{"channel":"trades","data":[{"coin":"ETH","tid":1}]}`)
	s.push(t, `{"channel":"trades","data":[{"coin":"BTC","tid":2}]}`)
	s.push(t, `{"channel":"trades","data":[{"coin":"BTC","tid":3}]}`)
	s.push(t, `{"channel":"candle","data":{"s":"BTC","i":"5m","n":1}}`)
	s.push(t, `{"channel":"candle","data":{"s":"BTC","i":"1m","n":2}}`)
	s.push(t, `{"channel":"clearinghouseState","data":{"dex":"xyz","user":"0x00000000000000000000000000000000000000aa"}}`)
	s.push(t, `{"channel":"clearinghouseState","data":{"dex":"","user":"0x00000000000000000000000000000000000000aa","clearinghouseState":{"time":7}}}`)

	if got := next(t, ethCh); got[0].TID != 1 {
		t.Errorf("ETH got tid %d", got[0].TID)
	}
	for _, c := range []chan []Trade{btcACh, btcBCh} {
		if a, b := next(t, c), next(t, c); a[0].TID != 2 || b[0].TID != 3 {
			t.Errorf("BTC got tids %d, %d, want 2, 3 in order", a[0].TID, b[0].TID)
		}
	}
	if got := next(t, candleCh); got.Trades != 2 {
		t.Errorf("candle got n=%d, want the 1m candle", got.Trades)
	}
	if got := next(t, chCh); got.ClearinghouseState.Time != 7 {
		t.Errorf("clearinghouseState got %+v, want the main dex", got)
	}
	none(t, ethCh)
	none(t, candleCh)
	none(t, chCh)
}

func TestWebSocketSubscribeRejected(t *testing.T) {
	s := newWSServer(t)
	ws := dialTest(t, s)
	_, err := ws.Trades(t.Context(), TradesSubscription{Coin: "BAD"}, func([]Trade) {})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !strings.HasPrefix(apiErr.Message, "Invalid subscription") {
		t.Fatalf("got %v, want the server's rejection", err)
	}
}

func TestWebSocketUnsubscribe(t *testing.T) {
	s := newWSServer(t)
	ws := dialTest(t, s)
	ctx := t.Context()
	h, ch := collect[[]Trade]()
	a, err := ws.Trades(ctx, TradesSubscription{Coin: "BTC"}, h)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ws.Trades(ctx, TradesSubscription{Coin: "BTC"}, func([]Trade) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Unsubscribe(ctx); err != nil {
		t.Fatal(err)
	}
	s.push(t, `{"channel":"trades","data":[{"coin":"BTC"}]}`)
	none(t, ch)
	if len(s.received()) != 1 {
		t.Fatalf("unsubscribed while another subscription shares it: %q", s.received())
	}
	if err := b.Unsubscribe(ctx); err != nil {
		t.Fatal(err)
	}
	if got := s.received(); len(got) != 2 || !strings.HasPrefix(got[1], "unsubscribe ") {
		t.Fatalf("server got %q, want an unsubscribe after the last one", got)
	}
	<-a.Done()
	if a.Err() != nil {
		t.Fatalf("Err after Unsubscribe = %v", a.Err())
	}
}

func TestWebSocketSubscriptionOverflow(t *testing.T) {
	s := newWSServer(t)
	ws := dialTest(t, s)
	release := make(chan struct{})
	defer close(release)
	sub, err := ws.Trades(t.Context(), TradesSubscription{Coin: "BTC"}, func([]Trade) { <-release })
	if err != nil {
		t.Fatal(err)
	}
	for range maxSubscriptionQueue + 2 { // one in the handler, a full queue, one more
		s.push(t, `{"channel":"trades","data":[{"coin":"BTC"}]}`)
	}
	select {
	case <-sub.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("subscription not ended by a full queue")
	}
	if !errors.Is(sub.Err(), ErrSubscriptionOverflow) {
		t.Fatalf("Err = %v, want ErrSubscriptionOverflow", sub.Err())
	}
	deadline := time.Now().Add(5 * time.Second)
	for got := s.received(); len(got) != 2 || !strings.HasPrefix(got[1], "unsubscribe "); got = s.received() {
		if time.Now().After(deadline) {
			t.Fatalf("server got %q, want an unsubscribe", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestWebSocketResubscribe(t *testing.T) {
	s := newWSServer(t)
	ws := dialTest(t, s)
	h, ch := collect[L2Book]()
	if _, err := ws.L2Book(t.Context(), L2BookSubscription{Coin: "BTC"}, h); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	_ = s.conn.CloseNow()
	s.mu.Unlock()
	<-s.connected

	deadline := time.Now().Add(5 * time.Second)
	for len(s.received()) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("no resubscribe after reconnect")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := s.received(); got[0] != got[1] {
		t.Fatalf("resubscribed with %s, want %s", got[1], got[0])
	}
	s.push(t, `{"channel":"l2Book","data":{"coin":"BTC","time":1}}`)
	if got := next(t, ch); got.Time != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestWebSocketPongTimeout(t *testing.T) {
	s := newWSServer(t)
	s.mutePings.Store(true)
	dialTest(t, s, func(c *wsConfig) { c.pingInterval, c.pongTimeout = 10*time.Millisecond, 10*time.Millisecond })
	select {
	case <-s.connected:
	case <-time.After(5 * time.Second):
		t.Fatal("no reconnect after an unanswered ping")
	}
}

func TestWebSocketPost(t *testing.T) {
	s := newWSServer(t)
	ws := dialTest(t, s)
	ctx := t.Context()

	mids, err := NewInfoClient(s.network(), WithWebSocket(ws)).AllMids(ctx, AllMidsRequest{})
	if err != nil || mids["BTC"] != "100.5" {
		t.Fatalf("AllMids = %v, %v", mids, err)
	}
	_, err = NewInfoClient(s.network(), WithWebSocket(ws)).Meta(ctx, MetaRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "unknown request" {
		t.Fatalf("Meta error = %v, want the server's error", err)
	}
	raw, err := ws.request(ctx, "exchange", []byte(`{"action":{}}`))
	if err != nil || decodeExchangeResponse(raw, nil) != nil {
		t.Fatalf("exchange post = %s, %v", raw, err)
	}
}

func TestWebSocketClose(t *testing.T) {
	s := newWSServer(t)
	before := runtime.NumGoroutine()
	ws := dialTest(t, s)
	sub, err := ws.Trades(t.Context(), TradesSubscription{Coin: "BTC"}, func([]Trade) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Close(); err != nil {
		t.Fatal(err)
	}
	<-sub.Done()
	if !errors.Is(sub.Err(), ErrWebSocketClosed) {
		t.Errorf("Err = %v, want ErrWebSocketClosed", sub.Err())
	}
	if _, err := ws.Bbo(t.Context(), BboSubscription{Coin: "BTC"}, func(BboEvent) {}); !errors.Is(err, ErrWebSocketClosed) {
		t.Errorf("subscribe after Close = %v", err)
	}
	if _, err := ws.request(t.Context(), "info", []byte(`{}`)); !errors.Is(err, ErrWebSocketClosed) {
		t.Errorf("post after Close = %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			t.Fatalf("goroutines leaked: %d > %d\n%s", runtime.NumGoroutine(), before, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestFastAssetCtxsUnmarshal(t *testing.T) {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestSpeed)
	_, _ = w.Write([]byte(`{"BTC":{"markPx":"100","midPx":"100.5"},"ETH":{"midPx":null}}`))
	_ = w.Close()
	frame, _ := json.Marshal(base64.StdEncoding.EncodeToString(buf.Bytes()))

	var got FastAssetCtxs
	if err := json.Unmarshal(frame, &got); err != nil {
		t.Fatal(err)
	}
	if btc := got["BTC"]; *btc.MarkPx != "100" || *btc.MidPx != "100.5" {
		t.Errorf("BTC = %+v", btc)
	}
	if eth, ok := got["ETH"]; !ok || eth.MarkPx != nil || eth.MidPx != nil {
		t.Errorf("ETH = %+v, %v", eth, ok)
	}
}
