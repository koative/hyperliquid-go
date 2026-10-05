package hyperliquid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// wsLiveCase is a subscription method under test: TestLiveWebSocket
// subscribes through it, TestWebSocketFixtures decodes its recorded first
// message into the type its handler receives.
type wsLiveCase struct {
	name     string // wire type, plus a variant suffix after "/"
	channel  string // message channel, if it differs from the wire type
	req      any
	explorer bool   // needs a client from DialExplorerWebSocket
	quiet    string // why the channel may emit nothing during the test
	sub      func(ctx context.Context, c *WebSocketClient, got func()) (*Subscription, error)
	decode   func(json.RawMessage) error
}

func wsDecode[T any](b json.RawMessage) error {
	var v T
	return unmarshal(b, &v)
}

// wsReqCase builds a case for a method with a request.
func wsReqCase[R, T any](name string, req R, m func(*WebSocketClient, context.Context, R, func(T)) (*Subscription, error)) wsLiveCase {
	return wsLiveCase{
		name: name, req: req, decode: wsDecode[T],
		sub: func(ctx context.Context, c *WebSocketClient, got func()) (*Subscription, error) {
			return m(c, ctx, req, func(T) { got() })
		},
	}
}

// wsCase builds a case for a method without a request.
func wsCase[T any](name string, m func(*WebSocketClient, context.Context, func(T)) (*Subscription, error)) wsLiveCase {
	return wsLiveCase{
		name: name, req: noParams{}, decode: wsDecode[T],
		sub: func(ctx context.Context, c *WebSocketClient, got func()) (*Subscription, error) {
			return m(c, ctx, func(T) { got() })
		},
	}
}

func (c wsLiveCase) on(channel string) wsLiveCase  { c.channel = channel; return c }
func (c wsLiveCase) quietly(why string) wsLiveCase { c.quiet = why; return c }
func (c wsLiveCase) onExplorer() wsLiveCase        { c.explorer = true; return c }

// wsCases lists every subscription method. user is an active trader, so
// its order updates and fills flow within seconds.
func wsCases(user Address) []wsLiveCase {
	return []wsLiveCase{
		wsReqCase("allMids", AllMidsSubscription{}, (*WebSocketClient).AllMids),
		wsReqCase("assetCtxs", AssetCtxsSubscription{}, (*WebSocketClient).AssetCtxs),
		wsCase("allDexsAssetCtxs", (*WebSocketClient).AllDexsAssetCtxs),
		wsCase("fastAssetCtxs", (*WebSocketClient).FastAssetCtxs),
		wsCase("spotAssetCtxs", (*WebSocketClient).SpotAssetCtxs),
		wsReqCase("activeAssetCtx", ActiveAssetCtxSubscription{Coin: "BTC"}, (*WebSocketClient).ActiveAssetCtx),
		wsReqCase("activeAssetCtx/spot", ActiveAssetCtxSubscription{Coin: "@107"}, (*WebSocketClient).ActiveSpotAssetCtx).on("activeSpotAssetCtx"),
		wsReqCase("activeAssetData", ActiveAssetDataSubscription{Coin: "BTC", User: user}, (*WebSocketClient).ActiveAssetData),
		wsReqCase("l2Book", L2BookSubscription{Coin: "BTC"}, (*WebSocketClient).L2Book),
		wsReqCase("bbo", BboSubscription{Coin: "BTC"}, (*WebSocketClient).Bbo),
		wsReqCase("trades", TradesSubscription{Coin: "BTC"}, (*WebSocketClient).Trades),
		wsReqCase("candle", CandleSubscription{Coin: "BTC", Interval: Candle1m}, (*WebSocketClient).Candle),
		wsReqCase("clearinghouseState", ClearinghouseStateSubscription{User: user}, (*WebSocketClient).ClearinghouseState),
		wsReqCase("allDexsClearinghouseState", AllDexsClearinghouseStateSubscription{User: user}, (*WebSocketClient).AllDexsClearinghouseState),
		wsReqCase("spotState", SpotStateSubscription{User: user}, (*WebSocketClient).SpotState),
		wsReqCase("openOrders", OpenOrdersSubscription{User: user}, (*WebSocketClient).OpenOrders),
		wsReqCase("twapStates", TwapStatesSubscription{User: user}, (*WebSocketClient).TwapStates),
		wsReqCase("orderUpdates", OrderUpdatesSubscription{User: user}, (*WebSocketClient).OrderUpdates).
			quietly("pushes only when one of the user's orders changes; no snapshot"),
		wsReqCase("userEvents", UserEventsSubscription{User: user}, (*WebSocketClient).UserEvents).on("user").
			quietly("pushes only on the user's fills, funding or liquidations; no snapshot"),
		wsReqCase("userFills", UserFillsSubscription{User: user}, (*WebSocketClient).UserFills),
		wsReqCase("userFundings", UserFundingsSubscription{User: user}, (*WebSocketClient).UserFundings),
		wsReqCase("userNonFundingLedgerUpdates", UserNonFundingLedgerUpdatesSubscription{User: user}, (*WebSocketClient).UserNonFundingLedgerUpdates),
		wsReqCase("userHistoricalOrders", UserHistoricalOrdersSubscription{User: user}, (*WebSocketClient).UserHistoricalOrders),
		wsReqCase("userTwapHistory", UserTwapHistorySubscription{User: user}, (*WebSocketClient).UserTwapHistory),
		wsReqCase("userTwapSliceFills", UserTwapSliceFillsSubscription{User: user}, (*WebSocketClient).UserTwapSliceFills),
		wsReqCase("notification", NotificationSubscription{User: user}, (*WebSocketClient).Notification).
			quietly("pushes only when the web app would notify the user"),
		wsReqCase("webData3", WebData3Subscription{User: user}, (*WebSocketClient).WebData3),
		wsCase("outcomeMetaUpdates", (*WebSocketClient).OutcomeMetaUpdates).
			quietly("pushes only when outcome market metadata changes"),
		wsCase("explorerBlock", (*WebSocketClient).ExplorerBlock).onExplorer(),
		wsCase("explorerTxs", (*WebSocketClient).ExplorerTxs).onExplorer(),
	}
}

// wsTransient reports whether err may pass on retry: anything but an
// answer from the API, except rate limits and server errors.
func wsTransient(err error) bool {
	var api *APIError
	return !errors.As(err, &api) || api.StatusCode == http.StatusTooManyRequests || api.StatusCode >= http.StatusInternalServerError
}

// wsRetry runs f up to three times while it fails transiently.
func wsRetry(t *testing.T, what string, f func(ctx context.Context) error) {
	t.Helper()
	var err error
	for i := range 3 {
		if i > 0 {
			time.Sleep(time.Duration(i) * 2 * time.Second)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = f(ctx)
		cancel()
		if err == nil || !wsTransient(err) {
			break
		}
		t.Logf("%s: %v (retrying)", what, err)
	}
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

func TestLiveWebSocket(t *testing.T) {
	requireLive(t)
	onErr := func(err error) {
		if strings.Contains(err.Error(), "decode") {
			t.Errorf("websocket: %v", err)
		} else {
			t.Logf("websocket: %v", err)
		}
	}
	dial := func(t *testing.T, network Network, explorer bool) *WebSocketClient {
		d := DialWebSocket
		if explorer {
			d = DialExplorerWebSocket
		}
		var c *WebSocketClient
		wsRetry(t, "dial", func(ctx context.Context) (err error) {
			c, err = d(ctx, network, WithWebSocketErrorHandler(onErr))
			return err
		})
		t.Cleanup(func() { _ = c.Close() })
		return c
	}
	ws, explorer := dial(t, Mainnet, false), dial(t, Mainnet, true)
	// Raw messages come from separate connections so that the typed
	// subscriptions still receive their snapshots.
	var rawWS, rawExplorer *WebSocketClient
	if recording() {
		rawWS, rawExplorer = dial(t, Mainnet, false), dial(t, Mainnet, true)
	}
	user := activeTrader(t, NewInfoClient(Mainnet))

	for _, tc := range wsCases(user) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, rc := ws, rawWS
			if tc.explorer {
				c, rc = explorer, rawExplorer
			}
			got := make(chan struct{})
			var once sync.Once
			var sub *Subscription
			wsRetry(t, "subscribe", func(ctx context.Context) (err error) {
				sub, err = tc.sub(ctx, c, func() { once.Do(func() { close(got) }) })
				return err
			})
			var raw chan json.RawMessage
			if rc != nil {
				typ, _, _ := strings.Cut(tc.name, "/")
				channel := tc.channel
				if channel == "" {
					channel = typ
				}
				raw = make(chan json.RawMessage, 1)
				var rsub *Subscription
				wsRetry(t, "subscribe raw", func(ctx context.Context) (err error) {
					rsub, err = subscribe(ctx, rc, typ, channel, tc.req, nil, func(b json.RawMessage) {
						select {
						case raw <- b:
						default:
						}
					})
					return err
				})
				defer func() { _ = rsub.Unsubscribe(context.Background()) }()
			}

			wait, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			select {
			case <-got:
			case <-wait.Done():
				if tc.quiet == "" {
					t.Fatal("no message within 20s")
				}
				t.Logf("no message within 20s; subscription confirmed (%s)", tc.quiet)
			}
			if raw != nil {
				select {
				case b := <-raw:
					writeFixture(t, "ws", tc.name, b)
				case <-wait.Done():
					if tc.quiet == "" {
						t.Error("no raw message within 20s")
					}
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := sub.Unsubscribe(ctx); err != nil {
				t.Errorf("unsubscribe: %v", err)
			}
		})
	}

	t.Run("post/info", func(t *testing.T) {
		t.Parallel()
		info := NewInfoClient(Mainnet, WithWebSocket(ws))
		wsRetry(t, "meta", func(ctx context.Context) error {
			meta, err := info.Meta(ctx, MetaRequest{})
			if err == nil && len(meta.Universe) == 0 {
				t.Error("meta: empty universe")
			}
			return err
		})
		wsRetry(t, "clearinghouseState", func(ctx context.Context) error {
			_, err := info.ClearinghouseState(ctx, ClearinghouseStateRequest{User: hlpVault})
			return err
		})
	})

	t.Run("post/exchange", func(t *testing.T) {
		t.Parallel()
		key := make([]byte, 32)
		_, _ = rand.Read(key)
		signer, err := NewPrivateKeySigner(hex.EncodeToString(key))
		if err != nil {
			t.Fatal(err)
		}
		ex := NewExchangeClient(Testnet, signer, WithWebSocket(dial(t, Testnet, false)))
		var api *APIError
		wsRetry(t, "updateLeverage", func(ctx context.Context) error {
			err := ex.UpdateLeverage(ctx, UpdateLeverageAction{Asset: 0, IsCross: true, Leverage: 1})
			if err == nil {
				return errors.New("succeeded for an account without funds")
			}
			if errors.As(err, &api) && !wsTransient(err) {
				return nil
			}
			return err
		})
		if !strings.Contains(strings.ToLower(api.Message), strings.ToLower(signer.Address().String())) {
			t.Errorf("updateLeverage: error %q does not name the signer %s", api.Message, signer.Address())
		}
	})
}

func TestWebSocketFixtures(t *testing.T) {
	for _, tc := range wsCases(Address{}) {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.decode(readFixture(t, "ws", tc.name)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
