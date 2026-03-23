-- +goose Up
CREATE TABLE clients (
    id               BIGSERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    email            TEXT,
    phone            TEXT,
    billing_address  TEXT,
    notes            TEXT,
    archived_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clients_slug ON clients(slug);

-- +goose Down
DROP TABLE clients;
