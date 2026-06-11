ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS recipient_name VARCHAR(120),
  ADD COLUMN IF NOT EXISTS phone VARCHAR(32),
  ADD COLUMN IF NOT EXISTS address_line VARCHAR(255),
  ADD COLUMN IF NOT EXISTS city VARCHAR(120),
  ADD COLUMN IF NOT EXISTS district VARCHAR(120),
  ADD COLUMN IF NOT EXISTS note TEXT,
  ADD COLUMN IF NOT EXISTS shipping_method VARCHAR(32),
  ADD COLUMN IF NOT EXISTS payment_method VARCHAR(32);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'orders_shipping_method_check'
  ) THEN
    ALTER TABLE orders
      ADD CONSTRAINT orders_shipping_method_check
      CHECK (shipping_method IS NULL OR shipping_method IN ('standard', 'express'));
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'orders_payment_method_check'
  ) THEN
    ALTER TABLE orders
      ADD CONSTRAINT orders_payment_method_check
      CHECK (payment_method IS NULL OR payment_method IN ('cod', 'demo_payment'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_orders_shipping_method ON orders(shipping_method);
CREATE INDEX IF NOT EXISTS idx_orders_payment_method ON orders(payment_method);
