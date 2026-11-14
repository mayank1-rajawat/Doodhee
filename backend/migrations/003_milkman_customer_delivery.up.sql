UPDATE users
SET approval_status = 'approved', updated_at = now()
WHERE role = 'seller' AND approval_status <> 'approved';

ALTER TABLE orders
  ADD COLUMN base_quantity numeric(10,3),
  ADD COLUMN approved_extra_quantity numeric(10,3) NOT NULL DEFAULT 0 CHECK(approved_extra_quantity >= 0),
  ADD COLUMN price_source varchar(20) NOT NULL DEFAULT 'default'
    CHECK(price_source IN ('default','customer','delivery'));

UPDATE orders SET base_quantity = quantity;
ALTER TABLE orders ALTER COLUMN base_quantity SET NOT NULL;
ALTER TABLE orders ADD CHECK(base_quantity > 0);

CREATE TABLE milkman_customers (
 id uuid PRIMARY KEY,
 milkman_id uuid NOT NULL REFERENCES users(id),
 customer_id uuid NOT NULL REFERENCES users(id),
 delivery_address text NOT NULL,
 is_active boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(milkman_id, customer_id),
 CHECK(length(trim(delivery_address)) >= 5),
 CHECK(milkman_id <> customer_id)
);

CREATE TABLE customer_product_prices (
 id uuid PRIMARY KEY,
 milkman_id uuid NOT NULL REFERENCES users(id),
 customer_id uuid NOT NULL REFERENCES users(id),
 product_id uuid NOT NULL REFERENCES products(id),
 price numeric(12,2) NOT NULL CHECK(price > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(milkman_id, customer_id, product_id)
);

CREATE TABLE extra_milk_requests (
 id uuid PRIMARY KEY,
 order_id uuid NOT NULL REFERENCES orders(id),
 customer_id uuid NOT NULL REFERENCES users(id),
 milkman_id uuid NOT NULL REFERENCES users(id),
 extra_quantity numeric(10,3) NOT NULL CHECK(extra_quantity > 0),
 status varchar(20) NOT NULL DEFAULT 'pending'
   CHECK(status IN ('pending','approved','rejected')),
 decision_note text NOT NULL DEFAULT '',
 decided_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX extra_milk_one_open_per_order
  ON extra_milk_requests(order_id)
  WHERE status IN ('pending','approved');

CREATE TABLE milkman_ratings (
 id uuid PRIMARY KEY,
 customer_id uuid NOT NULL REFERENCES users(id),
 milkman_id uuid NOT NULL REFERENCES users(id),
 rating smallint NOT NULL CHECK(rating BETWEEN 1 AND 5),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(customer_id, milkman_id),
 CHECK(customer_id <> milkman_id)
);

CREATE INDEX milkman_customers_milkman ON milkman_customers(milkman_id, is_active);
CREATE INDEX milkman_customers_customer ON milkman_customers(customer_id, is_active);
CREATE INDEX customer_prices_lookup ON customer_product_prices(milkman_id, customer_id, product_id);
CREATE INDEX extra_milk_milkman_status ON extra_milk_requests(milkman_id, status, created_at);
CREATE INDEX ratings_milkman ON milkman_ratings(milkman_id);
