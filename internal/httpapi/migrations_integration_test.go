//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"os"
	"testing"

	migrator "desafio-backend/internal/migrate"
)

func TestMain(m *testing.M) {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		if err := migrator.Up(context.Background(), u); err != nil {
			fmt.Fprintf(os.Stderr, "migrations: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}
