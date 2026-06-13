CREATE TABLE IF NOT EXISTS inventory_reservations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_reservations_user_expires_at
    ON inventory_reservations(user_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_inventory_reservations_product_expires_at
    ON inventory_reservations(product_id, expires_at);
