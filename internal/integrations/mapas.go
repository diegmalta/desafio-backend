package integrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MapasClient pings a map/geocoding health endpoint through a circuit breaker.
type MapasClient struct {
	base    string
	httpc   *http.Client
	breaker *gobreaker.CircuitBreaker

	mu          sync.Mutex
	lastPingErr string
	lastPingAt  time.Time
	cancel      context.CancelFunc
}

// NewMapasClient baseURL is prefix without trailing slash (…/mapas + /health). Empty returns nil.
func NewMapasClient(baseURL string, httpTimeout time.Duration) *MapasClient {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil
	}
	transport := otelhttp.NewTransport(http.DefaultTransport)
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:     "mapas",
		Timeout:  30 * time.Second,
		Interval: 60 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= 2
		},
	})
	return &MapasClient{
		base: base,
		httpc: &http.Client{
			Timeout:   httpTimeout,
			Transport: transport,
		},
		breaker: cb,
	}
}

// StartBackgroundPing runs periodic health checks until ctx is cancelled.
func (m *MapasClient) StartBackgroundPing(ctx context.Context, interval time.Duration) {
	if m == nil || m.base == "" {
		return
	}
	ctx, m.cancel = context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		m.pingOnce(context.Background())
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.pingOnce(context.Background())
			}
		}
	}()
}

// Stop ends background pings.
func (m *MapasClient) Stop() {
	if m == nil {
		return
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *MapasClient) pingOnce(ctx context.Context) {
	if m == nil || m.base == "" {
		return
	}
	u := m.base + "/health"
	_, err := m.breaker.Execute(func() (any, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := m.httpc.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("mapas health http %d", resp.StatusCode)
		}
		return nil, nil
	})
	m.mu.Lock()
	m.lastPingAt = time.Now()
	if err != nil {
		m.lastPingErr = err.Error()
	} else {
		m.lastPingErr = ""
	}
	m.mu.Unlock()
}

// StatusJSON returns fields for GET /mapas/status .
func (m *MapasClient) StatusJSON() map[string]any {
	if m == nil {
		return map[string]any{"circuit": "disabled", "reason": "nil_client"}
	}
	if m.base == "" {
		return map[string]any{"circuit": "disabled", "reason": "no_base_url"}
	}
	m.mu.Lock()
	lastErr := m.lastPingErr
	at := m.lastPingAt
	m.mu.Unlock()
	out := map[string]any{
		"circuit":       m.breaker.State().String(),
		"last_ping_at":  nil,
		"last_ping_err": nil,
	}
	if !at.IsZero() {
		out["last_ping_at"] = at.UTC().Format(time.RFC3339Nano)
	}
	if lastErr != "" {
		out["last_ping_err"] = lastErr
	}
	return out
}
