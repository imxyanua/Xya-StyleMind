CREATE TABLE IF NOT EXISTS user_addresses (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recipient_name VARCHAR(120) NOT NULL,
  phone VARCHAR(32) NOT NULL,
  address_line VARCHAR(255) NOT NULL,
  city VARCHAR(120) NOT NULL,
  district VARCHAR(120) NOT NULL,
  note TEXT,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (length(trim(recipient_name)) >= 2),
  CHECK (length(trim(phone)) >= 8),
  CHECK (length(trim(address_line)) >= 5),
  CHECK (length(trim(city)) >= 2),
  CHECK (length(trim(district)) >= 2)
);

CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_user_addresses_user_created_at ON user_addresses(user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_addresses_one_default
  ON user_addresses(user_id)
  WHERE is_default = TRUE;
