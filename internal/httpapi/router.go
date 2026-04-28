package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/integrations"
	"desafio-backend/internal/rdb"
	"desafio-backend/internal/webhook"
	"desafio-backend/internal/wsbus"
)

type Deps struct {
	Pool    *pgxpool.Pool
	Redis   *rdb.Client
	Webhook *webhook.Service
	AuthJWT gin.HandlerFunc

	Hub         *wsbus.Hub
	JWTSecret   string
	CPFPepper   string
	JWTIssuer   string
	JWTAudience string

	WSWriteTimeout time.Duration
	WSPingInterval time.Duration
	WSPongWait     time.Duration
	WSReadLimit    int64

	Chamados *integrations.ChamadosClient
	Mapas    *integrations.MapasClient
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
		touch := authjwt.TouchLastSeenMiddleware(deps.Pool)
		auth := deps.AuthJWT

		ngrp := r.Group("/notifications")
		ngrp.Use(auth, touch)
		ngrp.GET("/unread-count", handleUnreadCount(deps.Pool))
		ngrp.PATCH("/read-all", handleMarkAllRead(deps.Pool, deps.Chamados))
		ngrp.GET("/:id", handleGetNotification(deps.Pool))
		ngrp.GET("", handleListNotifications(deps.Pool))
		ngrp.PATCH("/:id/read", handleMarkRead(deps.Pool, deps.Chamados))

		cg := r.Group("/citizens")
		cg.Use(auth, touch)
		cg.GET("/me", handleCitizenMe(deps.Pool))

		dg := r.Group("/devices")
		dg.Use(auth, touch)
		dg.POST("", handleRegisterDevice(deps.Pool))
		dg.DELETE("", handleDeleteDevice(deps.Pool))

		chgrp := r.Group("/chamados")
		chgrp.Use(auth, touch)
		chgrp.GET("/:chamado_id/summary", handleChamadosSummary(deps.Pool, deps.Chamados))

		mgrp := r.Group("/mapas")
		mgrp.Use(auth, touch)
		mgrp.GET("/status", handleMapasStatus(deps.Mapas))
	} else {
		grp := r.Group("/notifications")
		{
			grp.GET("", stub501)
			grp.PATCH("/:id/read", stub501)
			grp.GET("/unread-count", stub501)
			grp.PATCH("/read-all", stub501)
			grp.GET("/:id", stub501)
		}
		r.Group("/citizens").GET("/me", stub501)
		r.Group("/devices").POST("", stub501)
		r.Group("/devices").DELETE("", stub501)
		r.Group("/chamados").GET("/:chamado_id/summary", stub501)
		r.Group("/mapas").GET("/status", stub501)
	}
	r.GET("/ws", handleWS(deps))
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
