DROP INDEX IF EXISTS idx_push_devices_citizen;
DROP TABLE IF EXISTS push_devices;
ALTER TABLE citizens DROP COLUMN IF EXISTS last_seen_at;
ALTER TABLE citizens DROP COLUMN IF EXISTS preferences;
