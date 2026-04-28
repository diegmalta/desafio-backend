package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsertPushDevice registers or updates a device token for push delivery.
func UpsertPushDevice(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID, platform, token string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO push_devices (citizen_id, platform, token, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		ON CONFLICT (citizen_id, token) DO UPDATE SET
			platform = EXCLUDED.platform,
			updated_at = now()
	`, citizenID, platform, token)
	return err
}

// DeletePushDevice removes a token for the citizen.
func DeletePushDevice(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID, token string) (int64, error) {
	tag, err := pool.Exec(ctx, `
		DELETE FROM push_devices WHERE citizen_id = $1 AND token = $2
	`, citizenID, token)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListPushTokensByCitizen returns FCM-style tokens for push fan-out.
func ListPushTokensByCitizen(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT token FROM push_devices WHERE citizen_id = $1`, citizenID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
