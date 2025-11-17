ALTER TABLE couriers
ADD COLUMN IF NOT EXISTS telegram_id;

ALTER TABLE couriers
RENAME COLUMN last_updated TO last_seen;

ALTER TABLE couriers
ADD COLUMN IF NOT EXISTS current_order_id;

ALTER TABLE couriers
DROP COLUMN IF EXISTS location POINT;