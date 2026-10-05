package hyperliquid

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTransport(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		wantStatus int // 0: success
		wantMsg    string
	}{
		{"json", http.StatusOK, `{"BTC":"1"}`, 0, ""},
		{"rate limited", http.StatusTooManyRequests, "null\n", http.StatusTooManyRequests, "null"},
		{"unprocessable", http.StatusUnprocessableEntity, "Failed to deserialize the JSON body", http.StatusUnprocessableEntity, "Failed to deserialize the JSON body"},
		{"ok but not json", http.StatusOK, "upstream error", http.StatusOK, "upstream error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath, gotType, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				gotPath, gotType, gotBody = r.URL.Path, r.Header.Get("Content-Type"), string(b)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			c := NewInfoClient(Network{APIURL: srv.URL})
			mids, err := c.AllMids(context.Background(), AllMidsRequest{})
			if gotPath != "/info" || gotType != "application/json" || gotBody != `{"type":"allMids"}` {
				t.Errorf("request: %s %q %s", gotPath, gotType, gotBody)
			}
			if tc.wantStatus == 0 {
				if err != nil || mids["BTC"] != "1" {
					t.Fatalf("got %v, %v", mids, err)
				}
				return
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tc.wantStatus || apiErr.Message != tc.wantMsg {
				t.Fatalf("got %#v", err)
			}
		})
	}
}
