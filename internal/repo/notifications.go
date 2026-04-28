package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
// When ok is true, chamadoID is the row's chamado_id for outbound sync.
func MarkNotificationRead(ctx context.Context, pool *pgxpool.Pool, citizenID, notificationID uuid.UUID) (ok bool, chamadoID string, err error) {
	err = pool.QueryRow(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE id = $1 AND citizen_id = $2 AND read_at IS NULL
		RETURNING chamado_id
	`, notificationID, citizenID).Scan(&chamadoID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	return true, chamadoID, nil
}

// GetNotificationByCitizen returns one notification if it belongs to the citizen.
func GetNotificationByCitizen(ctx context.Context, pool *pgxpool.Pool, citizenID, notificationID uuid.UUID) (*NotificationRow, error) {
	var r NotificationRow
	err := pool.QueryRow(ctx, `
		SELECT id, chamado_id, title, body, read_at, created_at,
			status_anterior, status_novo, event_type, source_timestamp
		FROM notifications
		WHERE id = $1 AND citizen_id = $2
	`, notificationID, citizenID).Scan(
		&r.ID, &r.ChamadoID, &r.Title, &r.Body, &r.ReadAt, &r.CreatedAt,
		&r.StatusAnterior, &r.StatusNovo, &r.EventType, &r.SourceTimestamp,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// CitizenHasNotificationForChamado returns true if the citizen has at least one notification for chamado_id.
func CitizenHasNotificationForChamado(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID, chamadoID string) (bool, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE citizen_id = $1 AND chamado_id = $2
	`, citizenID, chamadoID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// MarkAllNotificationsRead marks every unread notification for the citizen; returns chamado_id per row updated.
func MarkAllNotificationsRead(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID) ([]string, int64, error) {
	rows, err := pool.Query(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE citizen_id = $1 AND read_at IS NULL
		RETURNING chamado_id
	`, citizenID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var chamados []string
	for rows.Next() {
		var ch string
		if err := rows.Scan(&ch); err != nil {
			return nil, 0, err
		}
		chamados = append(chamados, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return chamados, int64(len(chamados)), nil
}

// CountUnreadNotifications returns how many unread notifications the citizen has.
func CountUnreadNotifications(ctx context.Context, pool *pgxpool.Pool, citizenID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE citizen_id = $1 AND read_at IS NULL
	`, citizenID).Scan(&n)
	return n, err
}
