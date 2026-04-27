-- Metadados do evento de webhook (auditoria; sem CPF em texto).

ALTER TABLE notifications
    ADD COLUMN status_anterior text,
    ADD COLUMN status_novo text,
    ADD COLUMN event_type text,
    ADD COLUMN source_timestamp timestamptz;
