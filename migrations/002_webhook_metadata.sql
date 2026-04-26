-- Metadados do evento de webhook (auditoria; sem CPF em texto).

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS status_anterior text,
    ADD COLUMN IF NOT EXISTS status_novo text,
    ADD COLUMN IF NOT EXISTS event_type text,
    ADD COLUMN IF NOT EXISTS source_timestamp timestamptz;
