package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OutboxRow is one pending or claimed outbox message.
type OutboxRow struct {
	ID            uuid.UUID
	CitizenID     uuid.UUID
	Payload       []byte
	Attempts      int
	NextAttemptAt time.Time
}

// InsertOutbox enqueues a JSON payload to be published after the notification commit.
func InsertOutbox(ctx context.Context, tx pgx.Tx, citizenID, notificationID uuid.UUID, payload []byte) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO event_outbox (citizen_id, notification_id, payload)
		VALUES ($1, $2, $3)
	`, citizenID, notificationID, payload)
	return err
}

// SelectPendingOutboxForUpdate returns up to batch rows locked for update (SKIP LOCKED).
// Caller must run inside a transaction.
func SelectPendingOutboxForUpdate(ctx context.Context, tx pgx.Tx, batch int) ([]OutboxRow, error) {
	if batch < 1 {
		batch = 1
	}
	rows, err := tx.Query(ctx, `
		SELECT id, citizen_id, payload, attempts, next_attempt_at
		FROM event_outbox
		WHERE status = 'pending' AND next_attempt_at <= now()
		ORDER BY created_at ASC, id ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OutboxRow
	for rows.Next() {
		var r OutboxRow
		if err := rows.Scan(&r.ID, &r.CitizenID, &r.Payload, &r.Attempts, &r.NextAttemptAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkOutboxSent marks a row as successfully published.
func MarkOutboxSent(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE event_outbox
		SET status = 'sent', sent_at = now(), last_error = NULL
		WHERE id = $1
	`, id)
	return err
}

// MarkOutboxPublishFailed increments attempts, sets last_error, next_attempt_at, and possibly status dead.
func MarkOutboxPublishFailed(ctx context.Context, tx pgx.Tx, id uuid.UUID, lastErr string, nextAttemptAt time.Time, dead bool) error {
	status := "pending"
	if dead {
		status = "dead"
	}
	_, err := tx.Exec(ctx, `
		UPDATE event_outbox
		SET attempts = attempts + 1,
		    last_error = $2,
		    next_attempt_at = $3,
		    status = $4
		WHERE id = $1
	`, id, lastErr, nextAttemptAt, status)
	return err
}
