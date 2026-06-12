CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    type VARCHAR(20) NOT NULL,
    value NUMERIC(12,2) NOT NULL,
    min_order_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    max_discount_amount NUMERIC(12,2),
    usage_limit INTEGER,
    used_count INTEGER NOT NULL DEFAULT 0,
    starts_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (code <> ''),
    CHECK (type IN ('percent', 'fixed')),
    CHECK (value > 0),
    CHECK (type <> 'percent' OR value <= 100),
    CHECK (min_order_amount >= 0),
    CHECK (max_discount_amount IS NULL OR max_discount_amount >= 0),
    CHECK (usage_limit IS NULL OR usage_limit > 0),
    CHECK (used_count >= 0),
    CHECK (usage_limit IS NULL OR used_count <= usage_limit),
    CHECK (expires_at IS NULL OR starts_at IS NULL OR expires_at > starts_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_coupons_code_lower ON coupons (LOWER(code));
CREATE INDEX IF NOT EXISTS idx_coupons_active_dates ON coupons (is_active, starts_at, expires_at);

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS subtotal_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS coupon_id UUID REFERENCES coupons(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS coupon_code VARCHAR(64);

UPDATE orders
SET subtotal_amount = total_amount
WHERE subtotal_amount = 0;

CREATE INDEX IF NOT EXISTS idx_orders_coupon_id ON orders(coupon_id);
