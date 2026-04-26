package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LookupCitizenID returns the citizen uuid for the fingerprint, or nil if none.
func LookupCitizenID(ctx context.Context, pool *pgxpool.Pool, fingerprint []byte) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM citizens WHERE fingerprint = $1`, fingerprint).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}
