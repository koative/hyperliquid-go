package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Live API checks call the real Hyperliquid API and run only when
// HYPERLIQUID_LIVE=1 (the nightly "Live API" workflow, or locally). Info
// and WebSocket checks use Mainnet and decode every response strictly;
// exchange checks submit every action to Testnet with a fresh key. With
// HYPERLIQUID_RECORD=1 they also rewrite the recorded responses under
// testdata/fixtures, which the offline fixture tests decode on every run.

func requireLive(t *testing.T) {
	t.Helper()
	if os.Getenv("HYPERLIQUID_LIVE") != "1" {
		t.Skip("set HYPERLIQUID_LIVE=1 to run live API checks")
	}
}

func recording() bool { return os.Getenv("HYPERLIQUID_RECORD") == "1" }

// hlpVault is the Hyperliquid Liquidity Provider vault, an always-active
// account on Mainnet.
var hlpVault = MustParseAddress("0xdfc24b077bc1425ad1dea75bcb6f8158e10df303")

// activeTrader returns the buyer of the latest BTC trade on Mainnet.
func activeTrader(t *testing.T, info *InfoClient) Address {
	t.Helper()
	trades, err := info.RecentTrades(context.Background(), RecentTradesRequest{Coin: "BTC"})
	if err != nil || len(trades) == 0 {
		t.Fatalf("RecentTrades: %v (%d trades)", err, len(trades))
	}
	return trades[0].Users[0]
}

// recordingTransport passes requests through and keeps the last response.
type recordingTransport struct {
	tr   transport
	last json.RawMessage
}

func (r *recordingTransport) request(ctx context.Context, endpoint string, body []byte) (json.RawMessage, error) {
	raw, err := r.tr.request(ctx, endpoint, body)
	r.last = raw
	return raw, err
}

// replayTransport answers every request with a recorded response.
type replayTransport json.RawMessage

func (r replayTransport) request(context.Context, string, []byte) (json.RawMessage, error) {
	return json.RawMessage(r), nil
}

func fixturePath(kind, name string) string {
	return filepath.Join("testdata", "fixtures", kind, name+".json")
}

// readFixture returns a recorded response, or skips if none was recorded.
func readFixture(t *testing.T, kind, name string) json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(fixturePath(kind, name))
	if os.IsNotExist(err) {
		t.Skipf("no fixture %s; record with HYPERLIQUID_LIVE=1 HYPERLIQUID_RECORD=1", fixturePath(kind, name))
	}
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// writeFixture stores raw, shrunk to keep the repository small: arrays keep
// their first maxFixtureItems elements and objects their first
// maxFixtureKeys keys (sorted), which preserves the shapes the decoder sees.
func writeFixture(t *testing.T, kind, name string, raw json.RawMessage) {
	t.Helper()
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("fixture %s/%s: %v", kind, name, err)
	}
	var buf bytes.Buffer
	writeShrunk(&buf, v, "")
	buf.WriteByte('\n')
	path := fixturePath(kind, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

const (
	maxFixtureItems = 8
	maxFixtureKeys  = 40
)

// writeShrunk writes v as indented JSON with sorted object keys.
func writeShrunk(buf *bytes.Buffer, v any, indent string) {
	in := indent + "  "
	switch v := v.(type) {
	case []any:
		if len(v) == 0 {
			buf.WriteString("[]")
			return
		}
		buf.WriteString("[\n")
		for i, e := range v[:min(len(v), maxFixtureItems)] {
			if i > 0 {
				buf.WriteString(",\n")
			}
			buf.WriteString(in)
			writeShrunk(buf, e, in)
		}
		buf.WriteString("\n" + indent + "]")
	case map[string]any:
		if len(v) == 0 {
			buf.WriteString("{}")
			return
		}
		keys := slices.Sorted(maps.Keys(v))
		buf.WriteString("{\n")
		for i, k := range keys[:min(len(keys), maxFixtureKeys)] {
			if i > 0 {
				buf.WriteString(",\n")
			}
			kb, _ := json.Marshal(k)
			buf.WriteString(in)
			buf.Write(kb)
			buf.WriteString(": ")
			writeShrunk(buf, v[k], in)
		}
		buf.WriteString("\n" + indent + "}")
	default:
		b, _ := json.Marshal(v)
		buf.Write(b)
	}
}
