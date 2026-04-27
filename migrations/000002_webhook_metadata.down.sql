ALTER TABLE notifications
    DROP COLUMN IF EXISTS status_anterior,
    DROP COLUMN IF EXISTS status_novo,
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS source_timestamp;
