CREATE TABLE IF NOT EXISTS order_assignments (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    courier_id INTEGER REFERENCES couriers(id),
    assigned_at TIMESTAMP DEFAULT NOW(),
    expired_at TIMESTAMP NOT NULL,
    courier_response_status TEXT NOT NULL DEFAULT 'waiting'
);