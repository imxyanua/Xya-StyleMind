CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    order_updates_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    payment_updates_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    return_updates_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    promotion_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_preferences_updated_at
    ON notification_preferences(updated_at DESC);
