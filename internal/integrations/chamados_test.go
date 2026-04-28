package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChamadosClientGetSummaryOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "1"})
	}))
	t.Cleanup(srv.Close)

	c := NewChamadosClient(srv.URL, 2*time.Second)
	body, status, err := c.GetSummary(context.Background(), "CH-X")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if len(body) == 0 {
		t.Fatal("empty body")
	}
}
