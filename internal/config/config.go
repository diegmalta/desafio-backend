package config

import (
	"os"
	"strconv"
	"time"
)

// Load reads required settings from the environment. Defaults suit local development.
func Load() Config {
	return Config{
		HTTPAddr:      getDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:   getDefault("DATABASE_URL", "postgres://notif:notif@localhost:5432/notif?sslmode=disable"),
		RedisAddr:     getDefault("REDIS_ADDR", "localhost:6379"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
		CPFPepper:     os.Getenv("CPF_PEPPER"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTIssuer:     os.Getenv("JWT_ISS"),
		JWTAudience:   os.Getenv("JWT_AUD"),

		OutboxBatchSize:    getIntDefault("OUTBOX_BATCH_SIZE", 50),
		OutboxPollInterval: getDurationDefault("OUTBOX_POLL_INTERVAL", 500*time.Millisecond),
		OutboxMaxAttempts:  getIntDefault("OUTBOX_MAX_ATTEMPTS", 5),
		OutboxBackoffBase:  getDurationDefault("OUTBOX_BACKOFF_BASE", 200*time.Millisecond),

		WSWriteTimeout: getDurationDefault("WS_WRITE_TIMEOUT", 10*time.Second),
		WSPingInterval: getDurationDefault("WS_PING_INTERVAL", 30*time.Second),
		WSPongWait:     getDurationDefault("WS_PONG_WAIT", 60*time.Second),
		WSReadLimit:    getInt64Default("WS_READ_LIMIT", 1<<20),

		OTELServiceName:    getDefault("OTEL_SERVICE_NAME", "desafio-backend"),
		OTELTracesExporter: getDefault("OTEL_TRACES_EXPORTER", "stdout"),

		ChamadosAPIBaseURL:    getDefault("CHAMADOS_API_BASE_URL", ""),
		MapasAPIBaseURL:       getDefault("MAPAS_API_BASE_URL", ""),
		PushWebhookURL:        getDefault("PUSH_WEBHOOK_URL", ""),
		HTTPClientTimeout:     getDurationDefault("HTTP_CLIENT_TIMEOUT", 5*time.Second),
		MapasPingInterval:     getDurationDefault("MAPAS_PING_INTERVAL", 20*time.Second),
		InternalUpstreamStubs: getBoolDefault("INTERNAL_UPSTREAM_STUBS", false),
	}
}

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	RedisAddr     string
	WebhookSecret string
	CPFPepper     string
	JWTSecret     string
	JWTIssuer     string
	JWTAudience   string

	OutboxBatchSize    int
	OutboxPollInterval time.Duration
	OutboxMaxAttempts  int
	OutboxBackoffBase  time.Duration

	WSWriteTimeout time.Duration
	WSPingInterval time.Duration
	WSPongWait     time.Duration
	WSReadLimit    int64

	OTELServiceName    string
	OTELTracesExporter string

	ChamadosAPIBaseURL string
	MapasAPIBaseURL    string
	PushWebhookURL     string
	HTTPClientTimeout  time.Duration
	MapasPingInterval  time.Duration

	InternalUpstreamStubs bool
}

func getDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func getInt64Default(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func getDurationDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

func getBoolDefault(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		return def
	}
}
