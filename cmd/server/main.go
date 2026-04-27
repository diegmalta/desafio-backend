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
	migrator "desafio-backend/internal/migrate"
	"desafio-backend/internal/notify"
	"desafio-backend/internal/rdb"
	"desafio-backend/internal/webhook"
	"desafio-backend/internal/wsbus"
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
	if err := migrator.Up(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("migrations: %v", err)
	}
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

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	hub := wsbus.NewHub()
	worker := &notify.Worker{
		Pool:         pgPool,
		Redis:        redisC.Inner,
		BatchSize:    cfg.OutboxBatchSize,
		PollInterval: cfg.OutboxPollInterval,
		MaxAttempts:  cfg.OutboxMaxAttempts,
		BackoffBase:  cfg.OutboxBackoffBase,
	}
	sub := &notify.Subscriber{Redis: redisC.Inner, Hub: hub}
	go worker.Run(rootCtx)
	go sub.Run(rootCtx)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	})
	wh := webhook.NewService(pgPool, cfg.WebhookSecret, cfg.CPFPepper)
	auth := authjwt.Middleware(pgPool, cfg.JWTSecret, cfg.CPFPepper, cfg.JWTIssuer, cfg.JWTAudience)
	httpapi.Register(router, &httpapi.Deps{
		Pool:           pgPool,
		Redis:          redisC,
		Webhook:        wh,
		AuthJWT:        auth,
		Hub:            hub,
		JWTSecret:      cfg.JWTSecret,
		CPFPepper:      cfg.CPFPepper,
		JWTIssuer:      cfg.JWTIssuer,
		JWTAudience:    cfg.JWTAudience,
		WSWriteTimeout: cfg.WSWriteTimeout,
		WSPingInterval: cfg.WSPingInterval,
		WSPongWait:     cfg.WSPongWait,
		WSReadLimit:    cfg.WSReadLimit,
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
	waitShutdown(srv, rootCancel, hub)
}

func waitShutdown(srv *http.Server, cancel context.CancelFunc, hub *wsbus.Hub) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Println("shutting down")
	if cancel != nil {
		cancel()
	}
	if hub != nil {
		hub.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
