DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'courier_response_status'
) THEN CREATE TYPE courier_response_status AS ENUM (
    'waiting',
    'accepted',
    'rejected',
    'expired'
);
END IF;
END $$;