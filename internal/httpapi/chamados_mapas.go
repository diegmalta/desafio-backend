package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/integrations"
	"desafio-backend/internal/repo"
)

func handleChamadosSummary(pool *pgxpool.Pool, ch *integrations.ChamadosClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ch == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"reason": "chamados_api_unconfigured"})
			return
		}
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		chamadoID := c.Param("chamado_id")
		if chamadoID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
			return
		}
		ok, err := repo.CitizenHasNotificationForChamado(c.Request.Context(), pool, *cid, chamadoID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		body, status, err := ch.GetSummary(c.Request.Context(), chamadoID)
		if err != nil {
			if errors.Is(err, gobreaker.ErrOpenState) {
				c.JSON(http.StatusServiceUnavailable, gin.H{"reason": "chamados_api_unavailable"})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": "chamados_upstream", "detail": err.Error()})
			return
		}
		if status != http.StatusOK {
			c.Data(status, "application/json", body)
			return
		}
		c.Data(http.StatusOK, "application/json", body)
	}
}

func handleMapasStatus(m *integrations.MapasClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil {
			c.JSON(http.StatusOK, gin.H{"circuit": "disabled", "reason": "nil_client"})
			return
		}
		c.JSON(http.StatusOK, m.StatusJSON())
	}
}
