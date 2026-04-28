package upstream

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegister_disabledNoRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, false)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_upstream/v1/chamados/x/summary", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 when disabled, got %d", w.Code)
	}
}

func TestRegister_chamadosSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_upstream/v1/chamados/CH-1/summary", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "CH-1") {
		t.Fatalf("body should include chamado id: %s", w.Body.String())
	}
}

func TestRegister_chamadosSummary_fail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_upstream/v1/chamados/CH-1/summary?fail=1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestRegister_mapasHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_upstream/mapas/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestRegister_mapasHealth_fail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_upstream/mapas/health?fail=1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
}

func TestRegister_citizenRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/_upstream/callbacks/citizen-read", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRegister_pushDispatch_ok(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/_upstream/v1/push/dispatch", strings.NewReader(`{"token":"t","title":"a","body":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRegister_pushDispatch_failHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, true)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/_upstream/v1/push/dispatch", strings.NewReader(`{"token":"t","title":"a","body":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Simulate-Fail", "1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d", w.Code)
	}
}
