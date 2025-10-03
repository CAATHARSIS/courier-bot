CREATE TYPE IF NOT EXISTS courier_response_status AS ENUM (
    'waiting',
    'accepted',
    'rejected',
    'expired'
);