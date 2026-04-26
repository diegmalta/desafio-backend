package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/rdb"
	"desafio-backend/internal/webhook"
)

type Deps struct {
	Pool    *pgxpool.Pool
	Redis   *rdb.Client
	Webhook *webhook.Service
	AuthJWT gin.HandlerFunc
}

// Register wires health, ready, and not-yet-implemented API routes.
func Register(r *gin.Engine, deps *Deps) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/ready", readyHandler(deps))

	if deps != nil && deps.Webhook != nil {
		r.POST("/webhook", deps.Webhook.HandlePOST)
	} else {
		r.POST("/webhook", stub501)
	}
	if deps != nil && deps.Pool != nil && deps.AuthJWT != nil {
		grp := r.Group("/notifications")
		grp.Use(deps.AuthJWT)
		grp.GET("", handleListNotifications(deps.Pool))
		grp.PATCH("/:id/read", handleMarkRead(deps.Pool))
		grp.GET("/unread-count", handleUnreadCount(deps.Pool))
	} else {
		grp := r.Group("/notifications")
		{
			grp.GET("", stub501)
			grp.PATCH("/:id/read", stub501)
			grp.GET("/unread-count", stub501)
		}
	}
	// WebSocket (upgrade) — 501 for now; real impl. em Fase 2
	r.GET("/ws", stub501)
}

func readyHandler(deps *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.Pool == nil || deps.Redis == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "missing dependencies"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if err := deps.Pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "db": err.Error()})
			return
		}
		if err := deps.Redis.Inner.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "redis": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

func stub501(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "not implemented",
		"status":  "phase_1_stub",
	})
}
