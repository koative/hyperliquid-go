package hyperliquid

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTupleMap(t *testing.T) {
	var m TupleMap[int, Decimal]
	if err := json.Unmarshal([]byte(`[[10,"1.5"],[9,"2"],[0,"0.1"]]`), &m); err != nil {
		t.Fatal(err)
	}
	if want := (TupleMap[int, Decimal]{10: "1.5", 9: "2", 0: "0.1"}); !reflect.DeepEqual(m, want) {
		t.Fatalf("decoded %v, want %v", m, want)
	}
	// Keys must be sorted numerically, not as text: signed actions depend on it.
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if want := `[[0,"0.1"],[9,"2"],[10,"1.5"]]`; string(b) != want {
		t.Fatalf("marshaled %s, want %s", b, want)
	}

	s, _ := json.Marshal(TupleMap[string, int]{"b": 1, "a": 2, "B": 3})
	if want := `[["B",3],["a",2],["b",1]]`; string(s) != want {
		t.Fatalf("marshaled %s, want %s", s, want)
	}
	if b, _ := json.Marshal(TupleMap[int, int](nil)); string(b) != "[]" {
		t.Fatalf("nil map marshaled %s, want []", b)
	}

	var p Portfolio
	if err := json.Unmarshal([]byte(`[["day",{"accountValueHistory":[[1700000000000,"12.5"]],"pnlHistory":[],"vlm":"0.0"}]]`), &p); err != nil {
		t.Fatal(err)
	}
	if got := p["day"].AccountValueHistory; len(got) != 1 || got[0] != (TimeValue{1700000000000, "12.5"}) {
		t.Fatalf("portfolio day history = %v", got)
	}
}
