CREATE TABLE IF NOT EXISTS event_outbox (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    citizen_id       uuid NOT NULL REFERENCES citizens (id) ON DELETE CASCADE,
    notification_id  uuid NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    payload          jsonb NOT NULL,
    status           text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'dead')),
    attempts         int NOT NULL DEFAULT 0,
    next_attempt_at  timestamptz NOT NULL DEFAULT now(),
    last_error       text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    sent_at          timestamptz
);

CREATE INDEX IF NOT EXISTS idx_event_outbox_pending
    ON event_outbox (next_attempt_at, created_at)
    WHERE status = 'pending';
