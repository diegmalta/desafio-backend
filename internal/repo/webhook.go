package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WebhookInsertParams holds one persisted notification row (CPF never stored).
type WebhookInsertParams struct {
	Fingerprint     []byte
	ChamadoID       string
	Title           string
	Body            string
	IdempotencyKey  string
	StatusAnterior  string
	StatusNovo      string
	EventType       string
	SourceTimestamp *time.Time
}

// EnsureCitizenID returns the citizen row id for the fingerprint (upsert by unique fingerprint).
func EnsureCitizenID(ctx context.Context, tx pgx.Tx, fingerprint []byte) (uuid.UUID, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO citizens (fingerprint) VALUES ($1)
		ON CONFLICT (fingerprint) DO NOTHING
	`, fingerprint)
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM citizens WHERE fingerprint = $1`, fingerprint).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// InsertNotificationIdempotent inserts a notification or detects duplicate idempotency_key.
// inserted is false when ON CONFLICT matched (same event re-delivered).
func InsertNotificationIdempotent(ctx context.Context, tx pgx.Tx, p WebhookInsertParams) (inserted bool, notificationID uuid.UUID, err error) {
	citizenID, err := EnsureCitizenID(ctx, tx, p.Fingerprint)
	if err != nil {
		return false, uuid.Nil, err
	}

	var nid uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO notifications (
			citizen_id, chamado_id, title, body, idempotency_key,
			status_anterior, status_novo, event_type, source_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`, citizenID, p.ChamadoID, p.Title, p.Body, p.IdempotencyKey,
		nullIfEmpty(p.StatusAnterior), nullIfEmpty(p.StatusNovo), nullIfEmpty(p.EventType), p.SourceTimestamp,
	).Scan(&nid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, uuid.Nil, nil
		}
		return false, uuid.Nil, err
	}
	return true, nid, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
