package upstream

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed fixtures/mapas_health.json
var mapasHealthBody []byte

func failQuery(c *gin.Context) bool {
	return c.Query("fail") == "1"
}

func failSimulate(c *gin.Context) bool {
	if failQuery(c) {
		return true
	}
	return c.GetHeader("X-Simulate-Fail") == "1"
}

func handleChamadosSummary(c *gin.Context) {
	if failQuery(c) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "simulated_upstream_failure"})
		return
	}
	id := c.Param("chamado_id")
	out := gin.H{
		"chamado_id": id,
		"resumo":     "resumo stub servido pelo processo local",
		"origem":     "internal/upstream",
	}
	c.JSON(http.StatusOK, out)
}

func handleCitizenRead(c *gin.Context) {
	if failSimulate(c) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "simulated_callback_failure"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleMapasHealth(c *gin.Context) {
	if failQuery(c) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "simulated_mapas_down"})
		return
	}
	c.Data(http.StatusOK, "application/json", append([]byte(nil), mapasHealthBody...))
}

func handlePushDispatch(c *gin.Context) {
	if failSimulate(c) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "simulated_push_failure"})
		return
	}
	var raw map[string]any
	if c.Request.ContentLength > 0 {
		_ = json.NewDecoder(c.Request.Body).Decode(&raw)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "received_fields": len(raw)})
}
