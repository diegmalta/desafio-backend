package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"desafio-backend/internal/authjwt"
	"desafio-backend/internal/config"
	"desafio-backend/internal/db"
	"desafio-backend/internal/httpapi"
	"desafio-backend/internal/rdb"
	"desafio-backend/internal/webhook"
)

func main() {
	cfg := config.Load()
	if cfg.WebhookSecret == "" || cfg.CPFPepper == "" {
		log.Fatal("WEBHOOK_SECRET and CPF_PEPPER must be set (see .env.example)")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set (see .env.example)")
	}

	ctx := context.Background()
	pgPool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pgPool.Close()

	redisC, err := rdb.Connect(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisC.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	})
	wh := webhook.NewService(pgPool, cfg.WebhookSecret, cfg.CPFPepper)
	auth := authjwt.Middleware(pgPool, cfg.JWTSecret, cfg.CPFPepper, cfg.JWTIssuer, cfg.JWTAudience)
	httpapi.Register(router, &httpapi.Deps{
		Pool:    pgPool,
		Redis:   redisC,
		Webhook: wh,
		AuthJWT: auth,
	})

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	waitShutdown(srv)
}

func waitShutdown(srv *http.Server) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
