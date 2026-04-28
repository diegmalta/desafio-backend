-- Preferências e atividade (sem CPF em claro); dispositivos para push mock/real.

ALTER TABLE citizens
    ADD COLUMN preferences jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN last_seen_at timestamptz;

CREATE TABLE push_devices (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    citizen_id  uuid NOT NULL REFERENCES citizens (id) ON DELETE CASCADE,
    platform    text NOT NULL,
    token       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (citizen_id, token)
);

CREATE INDEX idx_push_devices_citizen ON push_devices (citizen_id);
