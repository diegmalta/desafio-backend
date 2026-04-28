package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/repo"
)

type registerDeviceBody struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func handleRegisterDevice(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		var body registerDeviceBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		token := strings.TrimSpace(body.Token)
		platform := strings.TrimSpace(strings.ToLower(body.Platform))
		if token == "" || platform == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}
		if err := repo.UpsertPushDevice(c.Request.Context(), pool, *cid, platform, token); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleDeleteDevice(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := authjwt.CitizenID(c)
		if cid == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		token := strings.TrimSpace(c.Query("token"))
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_token"})
			return
		}
		n, err := repo.DeletePushDevice(c.Request.Context(), pool, *cid, token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if n == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
