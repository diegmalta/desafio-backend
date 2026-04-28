package upstream

import (
	"github.com/gin-gonic/gin"
)

// Register mounts HTTP handlers under /_upstream when enabled matches integration client paths.
func Register(r *gin.Engine, enabled bool) {
	if !enabled || r == nil {
		return
	}
	u := r.Group("/_upstream")
	u.GET("/v1/chamados/:chamado_id/summary", handleChamadosSummary)
	u.POST("/callbacks/citizen-read", handleCitizenRead)
	m := u.Group("/mapas")
	m.GET("/health", handleMapasHealth)
	u.POST("/v1/push/dispatch", handlePushDispatch)
}
