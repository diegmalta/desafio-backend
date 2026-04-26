//go:build integration

package webhook

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
	"testing"

	"github.com/gin-gonic/gin"

	"desafio-backend/internal/db"
)

func TestIntegration_webhookIdempotent(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set DATABASE_URL to run integration tests (e.g. postgres from docker compose)")
	}
	const whSecret = "integration-webhook-secret-32-chars______"
	const cpfPepper = "integration-cpf-pepper-32-chars__________"

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `TRUNCATE notifications, citizens CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	svc := NewService(pool, whSecret, cpfPepper)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhook", svc.HandlePOST)

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
	sig := signBody(raw, whSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte(`"duplicate"`)) {
		t.Fatal("first response should not be duplicate")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(raw))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Signature-256", "sha256="+sig)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second status=%d body=%s", w2.Code, w2.Body.String())
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte(`"duplicate":true`)) {
		t.Fatalf("expected duplicate, got %s", w2.Body.String())
	}
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
