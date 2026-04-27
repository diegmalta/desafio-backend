-- Schema mínimo (Fase 1). CPF nunca em texto: fingerprint derivada na Fase 2.
-- Requer PostgreSQL 13+ (gen_random_uuid no núcleo).

CREATE TABLE citizens (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint   bytea NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    citizen_id      uuid NOT NULL REFERENCES citizens (id) ON DELETE CASCADE,
    chamado_id      text NOT NULL,
    title           text NOT NULL,
    body            text NOT NULL,
    read_at         timestamptz,
    idempotency_key text UNIQUE,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_citizen ON notifications (citizen_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications (citizen_id) WHERE read_at IS NULL;
