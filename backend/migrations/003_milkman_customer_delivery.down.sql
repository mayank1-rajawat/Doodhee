DROP TABLE IF EXISTS milkman_ratings;
DROP TABLE IF EXISTS extra_milk_requests;
DROP TABLE IF EXISTS customer_product_prices;
DROP TABLE IF EXISTS milkman_customers;

ALTER TABLE orders
  DROP COLUMN IF EXISTS price_source,
  DROP COLUMN IF EXISTS approved_extra_quantity,
  DROP COLUMN IF EXISTS base_quantity;
