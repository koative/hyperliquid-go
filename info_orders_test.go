package hyperliquid

import (
	"context"
	"testing"
)

func TestOrderStatus(t *testing.T) {
	user := MustParseAddress("0x0000000000000000000000000000000000000001")
	cloid := MustParseCloid("0x00000000000000000000000000000abc")
	order := `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"1","sz":"0","oid":7,
		"timestamp":1,"origSz":"1","triggerCondition":"N/A","isTrigger":false,"triggerPx":"0","children":[],
		"isPositionTpsl":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc","cloid":null},
		"status":"filled","statusTimestamp":2}}`
	for _, tc := range []struct {
		name, resp, wantBody string
		ref                  OrderRef
		wantStatus           OrderProcessingStatus // "" means nil result
	}{
		{"oid", order, `{"type":"orderStatus","user":"0x0000000000000000000000000000000000000001","oid":7}`, OrderRef{Oid: 7}, "filled"},
		{"cloid", `{"status":"unknownOid"}`, `{"type":"orderStatus","user":"0x0000000000000000000000000000000000000001","oid":"0x00000000000000000000000000000abc"}`, OrderRef{Cloid: &cloid}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &recordTransport{resp: tc.resp}
			c := &InfoClient{tr: rt}
			got, err := c.OrderStatus(context.Background(), OrderStatusRequest{User: user, Oid: tc.ref})
			if err != nil {
				t.Fatal(err)
			}
			if string(rt.body) != tc.wantBody {
				t.Errorf("body = %s\nwant   %s", rt.body, tc.wantBody)
			}
			switch {
			case tc.wantStatus == "" && got != nil:
				t.Errorf("got %+v, want nil", got)
			case tc.wantStatus != "" && (got == nil || got.Status != tc.wantStatus || got.Order.Oid != 7):
				t.Errorf("got %+v, want status %q", got, tc.wantStatus)
			}
		})
	}
}
