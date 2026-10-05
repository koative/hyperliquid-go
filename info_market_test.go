package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestCandleSnapshotNestsReq(t *testing.T) {
	tr := &recordTransport{resp: `[]`}
	c := &InfoClient{tr: tr}
	if _, err := c.CandleSnapshot(context.Background(), CandleSnapshotRequest{Coin: "BTC", Interval: Candle1h, StartTime: 1}); err != nil {
		t.Fatal(err)
	}
	want := `{"type":"candleSnapshot","req":{"coin":"BTC","interval":"1h","startTime":1}}`
	if got := compact(t, tr.body); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestExplorerError(t *testing.T) {
	c := &InfoClient{explorer: &recordTransport{resp: `{"type":"error","message":"invalid block height: 0"}`}}
	_, err := c.TxDetails(context.Background(), TxDetailsRequest{Hash: "0x00"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "invalid block height: 0" {
		t.Errorf("err = %v, want APIError", err)
	}
}

func TestUnmarshalUnions(t *testing.T) {
	for _, tt := range []struct {
		in   string
		into any
		want any
	}{
		{`"question"`, new(OutcomeTemplateRole), &OutcomeTemplateRole{Question: true}},
		{
			`{"standaloneOutcome":{"sideNames":["Yes","No"]}}`, new(OutcomeTemplateRole),
			&OutcomeTemplateRole{StandaloneOutcome: &StandaloneOutcomeRole{SideNames: [2]string{"Yes", "No"}}},
		},
		{
			`{"questionOutcome":{"parent":"awardWinner"}}`, new(OutcomeTemplateRole),
			&OutcomeTemplateRole{QuestionOutcome: &QuestionOutcomeRole{Parent: "awardWinner"}},
		},
		{
			`[["setOracle",["0x0000000000000000000000000000000000000001"]],[{"hip3Star":"order"},[]]]`,
			new(TupleMap[SubDeployerVariant, []Address]), &TupleMap[SubDeployerVariant, []Address]{
				{Name: "setOracle"}:             {MustParseAddress("0x0000000000000000000000000000000000000001")},
				{Name: "order", HIP3Star: true}: {},
			},
		},
		{
			`["day",{"uptimeFraction":"1.0","predictedApr":"0.02","nSamples":1440}]`, new(ValidatorStats),
			&ValidatorStats{Period: "day", UptimeFraction: "1.0", PredictedAPR: "0.02", NSamples: 1440},
		},
	} {
		if err := json.Unmarshal([]byte(tt.in), tt.into); err != nil {
			t.Errorf("%s: %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(tt.into, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.in, tt.into, tt.want)
		}
	}
	if err := json.Unmarshal([]byte(`"bogus"`), new(OutcomeTemplateRole)); err == nil {
		t.Error("unknown role string accepted")
	}
}

func TestSubDeployerVariantRoundTrip(t *testing.T) {
	for _, in := range []string{`"setOracle"`, `{"hip3Star":"order"}`} {
		var v SubDeployerVariant
		if err := json.Unmarshal([]byte(in), &v); err != nil {
			t.Fatal(err)
		}
		if out, _ := json.Marshal(v); string(out) != in {
			t.Errorf("round trip %s = %s", in, out)
		}
	}
}
