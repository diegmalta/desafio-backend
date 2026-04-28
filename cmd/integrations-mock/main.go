// Command integrations-mock serves HTTP stubs for chamados, mapas health, and push (local/dev).
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	addr := ":8099"
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			addr = p
		} else {
			addr = ":" + p
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chamados/", handleChamadosSummary)
	mux.HandleFunc("/callbacks/citizen-read", handleCitizenRead)
	mux.HandleFunc("/mapas/health", handleMapasHealth)
	mux.HandleFunc("/v1/push/send", handlePushSend)
	log.Printf("integrations-mock listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleChamadosSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Query().Get("fail") == "1" {
		http.Error(w, "simulated upstream failure", http.StatusInternalServerError)
		return
	}
	// /v1/chamados/CH-xxx/summary
	path := strings.TrimPrefix(r.URL.Path, "/v1/chamados/")
	id := strings.TrimSuffix(path, "/summary")
	if id == "" || id == path {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"chamado_id": id,
		"resumo":     "mock summary from integrations-mock",
		"origem":     "stub",
	})
}

func handleCitizenRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("X-Simulate-Fail") == "1" {
		http.Error(w, "simulated callback failure", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func handleMapasHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Query().Get("fail") == "1" {
		http.Error(w, "mapas down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func handlePushSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Query().Get("fail") == "1" {
		http.Error(w, "push provider error", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
