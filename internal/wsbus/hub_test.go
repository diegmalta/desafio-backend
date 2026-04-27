package wsbus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestHub_Dispatch_deliversToMatchingCitizen(t *testing.T) {
	h := NewHub()
	defer h.Close()

	citizen := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		c := NewClient(h, citizen, conn, time.Second, 24*time.Hour, 24*time.Hour, 1<<20)
		go c.Run()
	}))
	t.Cleanup(srv.Close)

	u := strings.Replace(srv.URL, "http", "ws", 1)
	c1, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c1.Close() })

	time.Sleep(50 * time.Millisecond)
	h.Dispatch(citizen, []byte(`{"type":"notification"}`))

	_ = c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, b, err := c1.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"type":"notification"}` {
		t.Fatalf("got %s", string(b))
	}
}

func TestHub_Dispatch_otherCitizenGetsNothing(t *testing.T) {
	h := NewHub()
	defer h.Close()

	a := uuid.New()
	b := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		c := NewClient(h, a, conn, time.Second, 24*time.Hour, 24*time.Hour, 1<<20)
		go c.Run()
	}))
	t.Cleanup(srv.Close)

	u := strings.Replace(srv.URL, "http", "ws", 1)
	c1, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c1.Close() })
	time.Sleep(50 * time.Millisecond)

	h.Dispatch(b, []byte(`{"x":1}`))

	_ = c1.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = c1.ReadMessage()
	if err == nil {
		t.Fatal("expected timeout / no message for other citizen")
	}
}
