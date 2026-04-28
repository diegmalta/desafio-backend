package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ChamadosClient calls the external chamados HTTP API behind a circuit breaker.
type ChamadosClient struct {
	base    string
	httpc   *http.Client
	breaker *gobreaker.CircuitBreaker
}

// NewChamadosClient builds a client; baseURL should have no trailing slash. Empty base returns nil.
func NewChamadosClient(baseURL string, httpTimeout time.Duration) *ChamadosClient {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil
	}
	transport := otelhttp.NewTransport(http.DefaultTransport)
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:     "chamados",
		Timeout:  45 * time.Second,
		Interval: 60 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= 3
		},
	})
	return &ChamadosClient{
		base: base,
		httpc: &http.Client{
			Timeout:   httpTimeout,
			Transport: transport,
		},
		breaker: cb,
	}
}

// GetSummary fetches GET {base}/v1/chamados/{id}/summary .
func (c *ChamadosClient) GetSummary(ctx context.Context, chamadoID string) (body []byte, status int, err error) {
	if c == nil || c.base == "" {
		return nil, 0, errors.New("chamados client not configured")
	}
	u := c.base + "/v1/chamados/" + url.PathEscape(chamadoID) + "/summary"
	_, err = c.breaker.Execute(func() (any, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.httpc.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, rerr := io.ReadAll(resp.Body)
		body = b
		status = resp.StatusCode
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("chamados http %d", resp.StatusCode)
		}
		return nil, rerr
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			return nil, http.StatusServiceUnavailable, gobreaker.ErrOpenState
		}
		return body, status, err
	}
	return body, status, nil
}

// NotifyRead POSTs read acknowledgement to the chamados system (best-effort for caller).
func (c *ChamadosClient) NotifyRead(ctx context.Context, chamadoID, citizenRef string) error {
	if c == nil || c.base == "" {
		return nil
	}
	u := c.base + "/callbacks/citizen-read"
	payload := map[string]string{
		"chamado_id":  chamadoID,
		"citizen_ref": citizenRef,
		"event":       "notification_read",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.breaker.Execute(func() (any, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpc.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("chamados callback http %d", resp.StatusCode)
		}
		return nil, nil
	})
	if err != nil && errors.Is(err, gobreaker.ErrOpenState) {
		return gobreaker.ErrOpenState
	}
	return err
}
