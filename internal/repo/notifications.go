package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRow is one row returned to the REST API (no cross-citizen data).
type NotificationRow struct {
	ID              uuid.UUID
	ChamadoID       string
	Title           string
	Body            string
	ReadAt          *time.Time
	CreatedAt       time.Time
	StatusAnterior  *string
	StatusNovo      *string
	EventType       *string
	SourceTimestamp *time.Time
}

// ListNotificationsByCitizen returns a page and total count for the citizen.
func ListNotificationsByCitizen(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID, limit, offset int) ([]NotificationRow, int64, error) {
	var total int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE citizen_id = $1`, citizenID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, `
		SELECT id, chamado_id, title, body, read_at, created_at,
			status_anterior, status_novo, event_type, source_timestamp
		FROM notifications
		WHERE citizen_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, citizenID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []NotificationRow
	for rows.Next() {
		var r NotificationRow
		if err := rows.Scan(
			&r.ID, &r.ChamadoID, &r.Title, &r.Body, &r.ReadAt, &r.CreatedAt,
			&r.StatusAnterior, &r.StatusNovo, &r.EventType, &r.SourceTimestamp,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []NotificationRow{}
	}
	return out, total, nil
}

// MarkNotificationRead sets read_at if the row belongs to the citizen and is unread.
func MarkNotificationRead(ctx context.Context, pool *pgxpool.Pool, citizenID, notificationID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE id = $1 AND citizen_id = $2 AND read_at IS NULL
	`, notificationID, citizenID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CountUnreadNotifications returns how many unread notifications the citizen has.
func CountUnreadNotifications(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE citizen_id = $1 AND read_at IS NULL
	`, citizenID).Scan(&n)
	return n, err
}
