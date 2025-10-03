CREATE TABLE IF NOT EXISTS couriers (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    chat_id BIGINT,
    name TEXT NOT NULL,
    phone VARCHAR(10) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    last_seen TIMESTAMP WITH TIME ZONE,
    current_order_id INTEGER REFERENCES orders(id),
    rating DECIMAL(3, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()   
);