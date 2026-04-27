package notify

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"desafio-backend/internal/repo"
)

// Worker drains event_outbox and publishes to Redis.
type Worker struct {
	Pool         *pgxpool.Pool
	Redis        *redis.Client
	BatchSize    int
	PollInterval time.Duration
	MaxAttempts  int
	BackoffBase  time.Duration
}

// Run polls the outbox until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	if w.BatchSize < 1 {
		w.BatchSize = 1
	}
	if w.PollInterval <= 0 {
		w.PollInterval = 500 * time.Millisecond
	}
	if w.MaxAttempts < 1 {
		w.MaxAttempts = 5
	}
	if w.BackoffBase <= 0 {
		w.BackoffBase = 200 * time.Millisecond
	}

	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("notify worker: batch: %v", err)
			}
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) error {
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := repo.SelectPendingOutboxForUpdate(ctx, tx, w.BatchSize)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return tx.Commit(ctx)
	}

	for _, row := range rows {
		ch := CitizenChannel(row.CitizenID)
		pubErr := w.Redis.Publish(ctx, ch, string(row.Payload)).Err()
		if pubErr == nil {
			if err := repo.MarkOutboxSent(ctx, tx, row.ID); err != nil {
				return err
			}
			continue
		}
		newAttempts := row.Attempts + 1
		dead := newAttempts >= w.MaxAttempts
		next := time.Now().Add(w.nextBackoff(newAttempts))
		if dead {
			next = row.NextAttemptAt
		}
		if err := repo.MarkOutboxPublishFailed(ctx, tx, row.ID, pubErr.Error(), next, dead); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (w *Worker) nextBackoff(newAttempts int) time.Duration {
	shift := newAttempts
	if shift > 10 {
		shift = 10
	}
	return w.BackoffBase * time.Duration(uint(1)<<uint(shift))
}
