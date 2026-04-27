package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertWebhookDLQ records a webhook that failed after signature validation (best-effort).
func InsertWebhookDLQ(ctx context.Context, pool *pgxpool.Pool, rawBody []byte, signature, errorCode, errorMsg string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO webhook_dlq (raw_body, signature, error_code, error_msg)
		VALUES ($1, $2, $3, $4)
	`, rawBody, nullString(signature), errorCode, errorMsg)
	return err
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
