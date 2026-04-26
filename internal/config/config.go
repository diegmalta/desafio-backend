package config

import (
	"os"
)

// Load reads required settings from the environment. Defaults suit local development.
func Load() Config {
	return Config{
		HTTPAddr:      getDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:   getDefault("DATABASE_URL", "postgres://notif:notif@localhost:5432/notif?sslmode=disable"),
		RedisAddr:     getDefault("REDIS_ADDR", "localhost:6379"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
		CPFPepper:     os.Getenv("CPF_PEPPER"),
	}
}

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	RedisAddr     string
	WebhookSecret string
	CPFPepper     string
}

func getDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
