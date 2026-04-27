CREATE TABLE IF NOT EXISTS webhook_dlq (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    received_at  timestamptz NOT NULL DEFAULT now(),
    raw_body     bytea NOT NULL,
    signature    text,
    error_code   text NOT NULL,
    error_msg    text NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhook_dlq_received ON webhook_dlq (received_at DESC);
