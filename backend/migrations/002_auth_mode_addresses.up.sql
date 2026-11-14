ALTER TABLE auth_challenges
  ADD COLUMN auth_mode varchar(20) NOT NULL DEFAULT 'auto'
  CHECK(auth_mode IN ('auto','login','register'));

CREATE TABLE addresses (
 id uuid PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 label varchar(40) NOT NULL,
 address_line text NOT NULL,
 city varchar(100) NOT NULL,
 pincode varchar(12) NOT NULL,
 is_default boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX addresses_user ON addresses(user_id,created_at);
CREATE UNIQUE INDEX addresses_one_default ON addresses(user_id) WHERE is_default;
