-- +goose Up
CREATE TABLE settings (
    id                BIGSERIAL PRIMARY KEY,
    business_name     TEXT NOT NULL DEFAULT '',
    business_address  TEXT NOT NULL DEFAULT '',
    business_email    TEXT NOT NULL DEFAULT '',
    default_tax_rate  NUMERIC(5,2) NOT NULL DEFAULT 15.00,
    stripe_pk         TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings DEFAULT VALUES;

-- +goose Down
DROP TABLE settings;
