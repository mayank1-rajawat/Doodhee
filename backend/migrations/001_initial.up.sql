CREATE TABLE users (
 id uuid PRIMARY KEY, phone varchar(20) NOT NULL UNIQUE, full_name varchar(120) NOT NULL,
 role varchar(20) NOT NULL CHECK(role IN ('customer','seller','admin')),
 approval_status varchar(20) NOT NULL CHECK(approval_status IN ('pending','approved','rejected')),
 is_enabled boolean NOT NULL DEFAULT true, phone_verified_at timestamptz,
 approved_by uuid REFERENCES users(id), approved_at timestamptz, approval_reason text,
 address_line text NOT NULL DEFAULT '', city text NOT NULL DEFAULT '', pincode varchar(12) NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE products (
 id uuid PRIMARY KEY, seller_id uuid NOT NULL REFERENCES users(id), name varchar(120) NOT NULL,
 unit varchar(20) NOT NULL DEFAULT 'litre' CHECK(unit='litre'), price numeric(12,2) NOT NULL CHECK(price>0),
 daily_capacity numeric(10,3) NOT NULL CHECK(daily_capacity>0), is_available boolean NOT NULL DEFAULT true,
 admin_suspended boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE orders (
 id uuid PRIMARY KEY, customer_id uuid NOT NULL REFERENCES users(id), seller_id uuid NOT NULL REFERENCES users(id),
 product_id uuid NOT NULL REFERENCES products(id), service_date date NOT NULL,
 quantity numeric(10,3) NOT NULL CHECK(quantity>0), unit_price numeric(12,2) NOT NULL CHECK(unit_price>0),
 total_amount numeric(12,2) NOT NULL CHECK(total_amount>0),
 status varchar(30) NOT NULL CHECK(status IN ('pending','accepted','out_for_delivery','delivered','not_delivered','cancelled')),
 not_delivered_reason varchar(60), delivery_note text NOT NULL DEFAULT '', delivery_address text NOT NULL,
 customer_note text NOT NULL DEFAULT '', marked_by uuid REFERENCES users(id), delivery_marked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(status <> 'not_delivered' OR length(not_delivered_reason)>0)
);
CREATE TABLE auth_challenges (
 id uuid PRIMARY KEY, phone varchar(20) NOT NULL, audience text NOT NULL, purpose text NOT NULL,
 role text NOT NULL, code_hash text NOT NULL, expires_at timestamptz NOT NULL,
 attempt_count integer NOT NULL DEFAULT 0, consumed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE auth_sessions (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), audience text NOT NULL,
 refresh_token_hash text NOT NULL UNIQUE, token_family_id uuid NOT NULL, expires_at timestamptz NOT NULL,
 revoked_at timestamptz, last_seen_at timestamptz NOT NULL DEFAULT now(), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE mfa_methods (
 id uuid PRIMARY KEY, user_id uuid NOT NULL UNIQUE REFERENCES users(id), type text NOT NULL DEFAULT 'totp',
 secret_encrypted text NOT NULL, confirmed_at timestamptz NOT NULL DEFAULT now(), disabled_at timestamptz,
 last_used_step bigint NOT NULL DEFAULT 0
);
CREATE TABLE outbox_events (
 id uuid PRIMARY KEY, event_type text NOT NULL, aggregate_type text NOT NULL, aggregate_id uuid NOT NULL,
 payload jsonb NOT NULL, status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','published','failed')),
 attempt_count integer NOT NULL DEFAULT 0, last_error text, next_attempt_at timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz
);
CREATE TABLE admin_audit_logs (
 id uuid PRIMARY KEY, admin_id uuid NOT NULL REFERENCES users(id), action text NOT NULL,
 target_type text NOT NULL, target_id uuid NOT NULL, reason text NOT NULL, request_id text NOT NULL,
 before_data jsonb NOT NULL, after_data jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION deny_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Audit records are immutable'; END $$;
CREATE TRIGGER immutable_audit BEFORE UPDATE OR DELETE ON admin_audit_logs FOR EACH ROW EXECUTE FUNCTION deny_audit_mutation();
CREATE TABLE idempotency_keys (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), key text NOT NULL, route text NOT NULL,
 request_hash text NOT NULL, response jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(user_id,key,route)
);
CREATE TABLE notifications (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id), event_id uuid NOT NULL REFERENCES outbox_events(id),
 title text NOT NULL, body text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(user_id,event_id)
);
CREATE INDEX products_seller ON products(seller_id,is_available);
CREATE INDEX orders_seller ON orders(seller_id,service_date,status);
CREATE INDEX orders_customer ON orders(customer_id,service_date);
CREATE INDEX outbox_pending ON outbox_events(status,next_attempt_at);
CREATE INDEX challenges_phone ON auth_challenges(phone,created_at);
CREATE INDEX sessions_user ON auth_sessions(user_id);
