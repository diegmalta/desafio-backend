package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// PushHTTPClient sends push-like notifications to a mock or FCM-compatible HTTP endpoint.
type PushHTTPClient struct {
	url     string
	httpc   *http.Client
	breaker *gobreaker.CircuitBreaker
}

// NewPushHTTPClient posts JSON to pushURL (full URL, e.g. http://mock:8099/v1/push/send).
func NewPushHTTPClient(pushURL string, httpTimeout time.Duration) *PushHTTPClient {
	u := strings.TrimSpace(pushURL)
	if u == "" {
		return nil
	}
	transport := otelhttp.NewTransport(http.DefaultTransport)
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:     "push",
		Timeout:  20 * time.Second,
		Interval: 30 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= 5
		},
	})
	return &PushHTTPClient{
		url: u,
		httpc: &http.Client{
			Timeout:   httpTimeout,
			Transport: transport,
		},
		breaker: cb,
	}
}

// Send delivers one message to one device token.
func (p *PushHTTPClient) Send(ctx context.Context, token, title, body string) error {
	if p == nil {
		return nil
	}
	payload := map[string]string{
		"token": token,
		"title": title,
		"body":  body,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = p.breaker.Execute(func() (any, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := p.httpc.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("push http %d", resp.StatusCode)
		}
		return nil, nil
	})
	if errors.Is(err, gobreaker.ErrOpenState) {
		return nil
	}
	return err
}
