package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// recordTransport records the last request body and replies with resp.
type recordTransport struct {
	endpoint string
	body     []byte
	resp     string
}

func (r *recordTransport) request(_ context.Context, endpoint string, body []byte) (json.RawMessage, error) {
	r.endpoint, r.body = endpoint, body
	return json.RawMessage(r.resp), nil
}

func compact(t *testing.T, b []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

var (
	vectorDest  = MustParseAddress("0x5e9ee1089755c3435139848e47e6635505d5a13a")
	vectorVault = MustParseAddress("0x1719884eb866cb12b2287399b15f7db5e7d775ea")
	vectorOrder = OrderAction{Orders: []Order{{
		Asset: 4, IsBuy: true, Price: "1670.10", Size: "0.0147",
		Type: OrderType{Limit: &LimitOrder{Tif: TifIoc}},
	}}}
)

// TestActionEncodingMatchesPython checks that typed actions encode to the
// exact JSON (and therefore msgpack) the Python SDK signs.
func TestActionEncodingMatchesPython(t *testing.T) {
	v, _ := loadSigningVectors(t)
	want := map[string]json.RawMessage{}
	for _, tc := range v.L1 {
		want[tc.Name] = tc.Action
	}
	for _, tc := range v.UserSigned {
		if tc.Mainnet {
			want[tc.Name] = tc.Action
		}
	}
	cloid := MustParseCloid("0x00000000000000000000000000000001")
	for name, a := range map[string]Action{
		"order": vectorOrder,
		"order trigger cloid builder": OrderAction{
			Orders: []Order{{
				Asset: 10107, Price: "0.00012345", Size: "100000.000", ReduceOnly: true,
				Type:  OrderType{Trigger: &TriggerOrder{IsMarket: true, TriggerPx: "0.0001", Tpsl: TpslStopLoss}},
				Cloid: &cloid,
			}},
			Grouping: GroupingNormalTpsl,
			Builder:  &Builder{Address: vectorDest, Fee: 10},
		},
		"cancel":  CancelAction{Cancels: []Cancel{{Asset: 110000, Oid: 123456789012}}},
		"usdSend": UsdSendAction{Destination: vectorDest, Amount: "1.5"},
	} {
		got, err := encodeAction(a, 1700000000000, Mainnet)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if g, w := compact(t, got), compact(t, want[name]); g != w {
			t.Errorf("%s:\n got %s\nwant %s", name, g, w)
		}
	}
}

func TestMultiSigMatchesPython(t *testing.T) {
	v, signer := loadSigningVectors(t)
	actions := map[string]Action{
		"l1 order": vectorOrder,
		"usdSend":  UsdSendAction{Destination: vectorDest, Amount: "1"},
	}
	for _, tc := range v.MultiSig {
		rt := &recordTransport{resp: `{"status":"ok","response":{"type":"default"}}`}
		c := NewExchangeClient(Testnet, signer)
		c.tr = rt
		a := actions[tc.Name]

		inner, err := c.SignMultiSig(context.Background(), a, tc.MultiSigUser, tc.OuterSigner, tc.Nonce)
		if err != nil {
			t.Fatalf("%s: %v", tc.Name, err)
		}
		assertSignature(t, tc.Name+" inner", inner, tc.InnerSignature)

		if _, err := c.MultiSig(context.Background(), tc.MultiSigUser, a, tc.Nonce, []Signature{inner}); err != nil {
			t.Fatalf("%s: %v", tc.Name, err)
		}
		var req struct {
			Action    json.RawMessage
			Nonce     uint64
			Signature json.RawMessage
		}
		if err := json.Unmarshal(rt.body, &req); err != nil {
			t.Fatal(err)
		}
		if g, w := compact(t, req.Action), compact(t, tc.Action); g != w {
			t.Errorf("%s wrapper:\n got %s\nwant %s", tc.Name, g, w)
		}
		if req.Nonce != tc.Nonce {
			t.Errorf("%s: nonce %d, want %d", tc.Name, req.Nonce, tc.Nonce)
		}
		var sig Signature
		if err := json.Unmarshal(req.Signature, &sig); err != nil {
			t.Fatal(err)
		}
		assertSignature(t, tc.Name+" outer", sig, tc.Signature)
	}
}

func TestExchangeEnvelope(t *testing.T) {
	_, signer := loadSigningVectors(t)
	rt := &recordTransport{resp: `{"status":"ok","response":{"type":"default"}}`}
	c := NewExchangeClient(Mainnet, signer).WithVault(vectorVault)
	c.tr = rt

	if err := c.send(context.Background(), vectorOrder, 1700000000000, nil); err != nil {
		t.Fatal(err)
	}
	var req map[string]json.RawMessage
	if err := json.Unmarshal(rt.body, &req); err != nil {
		t.Fatal(err)
	}
	if rt.endpoint != "exchange" || string(req["vaultAddress"]) != `"`+vectorVault.String()+`"` {
		t.Errorf("L1 envelope: endpoint %q body %s", rt.endpoint, rt.body)
	}

	// User-signed actions never carry a vault.
	if err := c.UsdSend(context.Background(), UsdSendAction{Destination: vectorDest, Amount: "1"}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(rt.body, []byte("vaultAddress")) {
		t.Errorf("user-signed envelope has vaultAddress: %s", rt.body)
	}
}

func TestExchangeErrors(t *testing.T) {
	_, signer := loadSigningVectors(t)
	c := NewExchangeClient(Testnet, signer)
	rt := &recordTransport{}
	c.tr = rt

	rt.resp = `{"status":"err","response":"Insufficient margin"}`
	var apiErr *APIError
	if _, err := c.Order(context.Background(), vectorOrder); !errors.As(err, &apiErr) || apiErr.Message != "Insufficient margin" {
		t.Errorf("err status: got %v", err)
	}

	rt.resp = `{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":1}},{"error":"Price too far"},"waitingForFill",{"filled":{"totalSz":"0.1","avgPx":"10","oid":3}}]}}}`
	st, err := c.Order(context.Background(), vectorOrder)
	var se *StatusError
	if !errors.As(err, &se) || se.Index != 1 || se.Message != "Price too far" {
		t.Errorf("status error: got %v", err)
	}
	if len(st) != 4 || st[0].Resting.Oid != 1 || st[2].Status != "waitingForFill" || st[3].Filled.AvgPx != "10" {
		t.Errorf("statuses: %+v", st)
	}

	rt.resp = `{"status":"ok","response":{"type":"cancel","data":{"statuses":["success",{"error":"Order was never placed"}]}}}`
	if err := c.Cancel(context.Background(), CancelAction{}); !errors.As(err, &se) || se.Index != 1 {
		t.Errorf("cancel error: got %v", err)
	}
}

func TestNonceClockIsStrictlyIncreasing(t *testing.T) {
	var n nonceClock
	prev := n.next()
	for range 1000 {
		next := n.next()
		if next <= prev {
			t.Fatalf("nonce %d after %d", next, prev)
		}
		prev = next
	}
}
