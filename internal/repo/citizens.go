package repo

import (
	"context"
	"time"

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

// TouchCitizenLastSeen sets last_seen_at for activity tracking (no PII).
func TouchCitizenLastSeen(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE citizens SET last_seen_at = now() WHERE id = $1`, citizenID)
	return err
}

// CitizenMeRow is safe profile data for GET /citizens/me (no fingerprint exposure).
type CitizenMeRow struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	LastSeenAt  *time.Time
	Preferences []byte
}

// GetCitizenByID loads citizen row for /me.
func GetCitizenByID(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID) (*CitizenMeRow, error) {
	var r CitizenMeRow
	var lastSeen *time.Time
	err := pool.QueryRow(ctx, `
		SELECT id, created_at, last_seen_at, coalesce(preferences, '{}'::jsonb)
		FROM citizens WHERE id = $1
	`, citizenID).Scan(&r.ID, &r.CreatedAt, &lastSeen, &r.Preferences)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.LastSeenAt = lastSeen
	return &r, nil
}
