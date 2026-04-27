//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/db"
	migrator "desafio-backend/internal/migrate"
	"desafio-backend/internal/notify"
	"desafio-backend/internal/rdb"
	"desafio-backend/internal/webhook"
	"desafio-backend/internal/wsbus"
)

func TestIntegration_wsUnauthorized(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	redisAddr := os.Getenv("REDIS_ADDR")
	if databaseURL == "" || redisAddr == "" {
		t.Skip("set DATABASE_URL and REDIS_ADDR for integration tests")
	}
	ctx := context.Background()
	if err := migrator.Up(ctx, databaseURL); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(pool.Close)
	redisC, err := rdb.Connect(ctx, redisAddr)
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = redisC.Close() })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	hub := wsbus.NewHub()
	t.Cleanup(func() { hub.Close() })
	const jwtSecret = "integration-jwt-secret-32-bytes-minimum___"
	const cpfPepper = "integration-cpf-pepper-32-bytes-minimum____"
	Register(r, &Deps{
		Pool:           pool,
		Redis:          redisC,
		Hub:            hub,
		JWTSecret:      jwtSecret,
		CPFPepper:      cpfPepper,
		WSWriteTimeout: 10 * time.Second,
		WSPingInterval: 30 * time.Second,
		WSPongWait:     60 * time.Second,
		WSReadLimit:    1 << 20,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/ws")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without token, got %d", resp.StatusCode)
	}
}

func TestIntegration_wsWebhookBroadcast(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	redisAddr := os.Getenv("REDIS_ADDR")
	if databaseURL == "" || redisAddr == "" {
		t.Skip("set DATABASE_URL and REDIS_ADDR for integration tests")
	}
	const jwtSecret = "integration-jwt-secret-32-bytes-minimum___"
	const cpfPepper = "integration-cpf-pepper-32-bytes-minimum____"
	const webhookSecret = "integration-webhook-secret-32-chars______"

	ctx := context.Background()
	if err := migrator.Up(ctx, databaseURL); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(pool.Close)
	redisC, err := rdb.Connect(ctx, redisAddr)
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = redisC.Close() })

	_, err = pool.Exec(ctx, `TRUNCATE webhook_dlq, event_outbox, notifications, citizens CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	hub := wsbus.NewHub()
	t.Cleanup(func() { hub.Close() })

	worker := &notify.Worker{
		Pool:         pool,
		Redis:        redisC.Inner,
		BatchSize:    20,
		PollInterval: 100 * time.Millisecond,
		MaxAttempts:  5,
		BackoffBase:  50 * time.Millisecond,
	}
	sub := &notify.Subscriber{Redis: redisC.Inner, Hub: hub}
	go worker.Run(rootCtx)
	go sub.Run(rootCtx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	wh := webhook.NewService(pool, webhookSecret, cpfPepper)
	auth := authjwt.Middleware(pool, jwtSecret, cpfPepper, "", "")
	Register(r, &Deps{
		Pool:           pool,
		Redis:          redisC,
		Webhook:        wh,
		AuthJWT:        auth,
		Hub:            hub,
		JWTSecret:      jwtSecret,
		CPFPepper:      cpfPepper,
		WSWriteTimeout: 10 * time.Second,
		WSPingInterval: 30 * time.Second,
		WSPongWait:     60 * time.Second,
		WSReadLimit:    1 << 20,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	wsURL := strings.Replace(srv.URL, "http", "ws", 1) + "/ws"
	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+mintJWT(t, jwtSecret, "12345678901"))
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	wsConn, resp, err := dialer.Dial(wsURL, hdr)
	if err != nil {
		t.Fatalf("dial ws: %v (resp=%v)", err, resp)
	}
	t.Cleanup(func() { _ = wsConn.Close() })

	payload := map[string]any{
		"chamado_id":      "CH-2024-001234",
		"tipo":            "status_change",
		"cpf":             "12345678901",
		"status_anterior": "em_analise",
		"status_novo":     "em_execucao",
		"titulo":          "Buraco",
		"descricao":       "Equipe a caminho",
		"timestamp":       "2024-11-15T14:30:00Z",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	_, _ = mac.Write(raw)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/webhook", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+sig)
	hc := &http.Client{Timeout: 10 * time.Second}
	whResp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer whResp.Body.Close()
	if whResp.StatusCode != http.StatusOK {
		t.Fatalf("webhook status=%d", whResp.StatusCode)
	}

	_ = wsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws: %v", err)
	}
	var envelope struct {
		Type      string `json:"type"`
		ChamadoID string `json:"chamado_id"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("json: %v body=%s", err, string(msg))
	}
	if envelope.Type != "notification" || envelope.ChamadoID != "CH-2024-001234" {
		t.Fatalf("unexpected envelope: %+v raw=%s", envelope, string(msg))
	}
}
