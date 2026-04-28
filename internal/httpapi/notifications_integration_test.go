//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/db"
	"desafio-backend/internal/identity"
	"desafio-backend/internal/integrations"
	migrator "desafio-backend/internal/migrate"
	"desafio-backend/internal/rdb"
	"desafio-backend/internal/webhook"
	"desafio-backend/internal/wsbus"
)

func TestIntegration_notificationsJWT(t *testing.T) {
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

	_, err = pool.Exec(ctx, `TRUNCATE webhook_dlq, event_outbox, notifications, push_devices, citizens CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	cpfA := "12345678901"
	cpfB := "98765432100"
	fpA := identity.CitizenFingerprint([]byte(cpfPepper), cpfA)
	fpB := identity.CitizenFingerprint([]byte(cpfPepper), cpfB)

	var idA, idB uuid.UUID
	err = pool.QueryRow(ctx, `INSERT INTO citizens (fingerprint) VALUES ($1) RETURNING id`, fpA).Scan(&idA)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx, `INSERT INTO citizens (fingerprint) VALUES ($1) RETURNING id`, fpB).Scan(&idB)
	if err != nil {
		t.Fatal(err)
	}

	notifB := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO notifications (id, citizen_id, chamado_id, title, body, idempotency_key)
		VALUES ($1, $2, 'CH-B', 't', 'b', 'unique-b-1')
	`, notifB, idB)
	if err != nil {
		t.Fatal(err)
	}
	notifA := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO notifications (id, citizen_id, chamado_id, title, body, idempotency_key)
		VALUES ($1, $2, 'CH-A', 't', 'b', 'unique-a-1')
	`, notifA, idA)
	if err != nil {
		t.Fatal(err)
	}

	mockChamados := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/chamados/") && strings.HasSuffix(r.URL.Path, "/summary") {
			_ = json.NewEncoder(w).Encode(map[string]string{"stub": "chamados"})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/callbacks/citizen-read" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(mockChamados.Close)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	wh := webhook.NewService(pool, webhookSecret, cpfPepper)
	auth := authjwt.Middleware(pool, jwtSecret, cpfPepper, "", "")
	hub := wsbus.NewHub()
	t.Cleanup(func() { hub.Close() })
	Register(r, &Deps{
		Pool:           pool,
		Redis:          redisC,
		Webhook:        wh,
		AuthJWT:        auth,
		Hub:            hub,
		JWTSecret:      jwtSecret,
		CPFPepper:      cpfPepper,
		JWTIssuer:      "",
		JWTAudience:    "",
		WSWriteTimeout: 10 * time.Second,
		WSPingInterval: 30 * time.Second,
		WSPongWait:     60 * time.Second,
		WSReadLimit:    1 << 20,
		Chamados:       integrations.NewChamadosClient(mockChamados.URL, 5*time.Second),
	})

	tokenA := mintJWT(t, jwtSecret, cpfA)
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=10&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list A: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "CH-A") || strings.Contains(w.Body.String(), "CH-B") {
		t.Fatalf("list body should only contain CH-A: %s", w.Body.String())
	}

	reqPatch := httptest.NewRequest(http.MethodPatch, "/notifications/"+notifB.String()+"/read", nil)
	reqPatch.Header.Set("Authorization", "Bearer "+tokenA)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, reqPatch)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("citizen A patching B notification: want 404 got %d %s", w2.Code, w2.Body.String())
	}

	tokenB := mintJWT(t, jwtSecret, cpfB)
	reqPatch2 := httptest.NewRequest(http.MethodPatch, "/notifications/"+notifB.String()+"/read", nil)
	reqPatch2.Header.Set("Authorization", "Bearer "+tokenB)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, reqPatch2)
	if w3.Code != http.StatusNoContent {
		t.Fatalf("citizen B patch own: want 204 got %d %s", w3.Code, w3.Body.String())
	}

	reqGetOne := httptest.NewRequest(http.MethodGet, "/notifications/"+notifA.String(), nil)
	reqGetOne.Header.Set("Authorization", "Bearer "+tokenA)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, reqGetOne)
	if w4.Code != http.StatusOK || !strings.Contains(w4.Body.String(), "CH-A") {
		t.Fatalf("get one: %d %s", w4.Code, w4.Body.String())
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/citizens/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+tokenA)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, reqMe)
	if w5.Code != http.StatusOK || !strings.Contains(w5.Body.String(), idA.String()) {
		t.Fatalf("citizens me: %d %s", w5.Code, w5.Body.String())
	}

	reqSum := httptest.NewRequest(http.MethodGet, "/chamados/CH-A/summary", nil)
	reqSum.Header.Set("Authorization", "Bearer "+tokenA)
	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, reqSum)
	if w6.Code != http.StatusOK {
		t.Fatalf("chamados summary: %d %s", w6.Code, w6.Body.String())
	}
}

func mintJWT(t *testing.T, secret, cpf11 string) string {
	t.Helper()
	claims := struct {
		jwt.RegisteredClaims
		PreferredUsername string `json:"preferred_username"`
	}{
		PreferredUsername: cpf11,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}
