CREATE INDEX IF NOT EXISTS idx_products_price ON products(price);
CREATE INDEX IF NOT EXISTS idx_products_stock ON products(stock);
CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_products_lower_name ON products(LOWER(name));
CREATE INDEX IF NOT EXISTS idx_products_lower_description ON products(LOWER(description));
CREATE INDEX IF NOT EXISTS idx_product_reviews_product_rating ON product_reviews(product_id, rating);
