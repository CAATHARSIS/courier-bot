ALTER TABLE couriers
DROP COLUMN IF EXISTS telegram_id;

ALTER TABLE couriers
RENAME COLUMN last_seen TO last_updated;

ALTER TABLE couriers
DROP COLUMN IF EXISTS current_order_id;

ALTER TABLE couriers
ADD COLUMN IF NOT EXISTS location POINT;