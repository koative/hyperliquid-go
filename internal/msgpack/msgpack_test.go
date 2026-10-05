package msgpack

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// testdata/vectors.jsonl was produced with Python msgpack 1.x:
// {"json": doc, "msgpack": msgpack.packb(json.loads(doc)).hex()}.
func TestFromJSONMatchesPython(t *testing.T) {
	f, err := os.Open("testdata/vectors.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var v struct {
			JSON    string `json:"json"`
			Msgpack string `json:"msgpack"`
		}
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		got, err := FromJSON([]byte(v.JSON))
		if err != nil {
			t.Fatalf("FromJSON(%.60s): %v", v.JSON, err)
		}
		if h := hex.EncodeToString(got); h != v.Msgpack {
			t.Errorf("FromJSON(%.60s)\n got %s\nwant %s", v.JSON, h, v.Msgpack)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestFromJSONLargeContainers(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want []byte
	}{
		{65535, []byte{0xdc, 0xff, 0xff}},
		{65536, []byte{0xdd, 0x00, 0x01, 0x00, 0x00}},
	} {
		doc := "[" + strings.Repeat("0,", tc.n-1) + "0]"
		got, err := FromJSON([]byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(got, tc.want) || len(got) != len(tc.want)+tc.n {
			t.Errorf("array of %d: header % x, len %d", tc.n, got[:5], len(got))
		}
	}
}

func TestFromJSONRejects(t *testing.T) {
	for _, doc := range []string{`1.5`, `[1e3]`, `{"a":18446744073709551616}`, `{"a":1} {}`, `{"a":`} {
		if _, err := FromJSON([]byte(doc)); err == nil {
			t.Errorf("FromJSON(%s): want error", doc)
		}
	}
}

func FuzzFromJSON(f *testing.F) {
	f.Add([]byte(`{"type":"order","orders":[{"a":0,"b":true,"p":"1","s":"2","r":false}],"grouping":"na"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = FromJSON(data) // must not panic
	})
}
