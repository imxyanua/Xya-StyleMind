CREATE TABLE IF NOT EXISTS return_requests (
  id UUID PRIMARY KEY,
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'requested',
  admin_note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (length(trim(reason)) >= 10),
  CHECK (length(reason) <= 1000),
  CHECK (admin_note IS NULL OR length(admin_note) <= 1000),
  CHECK (status IN ('requested', 'approved', 'rejected', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_return_requests_user_created_at ON return_requests(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_return_requests_order_id ON return_requests(order_id);
CREATE INDEX IF NOT EXISTS idx_return_requests_status_created_at ON return_requests(status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_return_requests_active_order
  ON return_requests(order_id)
  WHERE status IN ('requested', 'approved');
