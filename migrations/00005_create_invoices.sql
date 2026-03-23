-- +goose Up
CREATE TYPE invoice_status AS ENUM (
    'draft',
    'sent',
    'viewed',
    'paid',
    'void'
);

CREATE TABLE invoices (
    id             BIGSERIAL PRIMARY KEY,
    number         TEXT NOT NULL UNIQUE,
    client_id      BIGINT NOT NULL REFERENCES clients(id),
    status         invoice_status NOT NULL DEFAULT 'draft',
    tax_rate       NUMERIC(5,2) NOT NULL,
    notes          TEXT,
    due_date       DATE,
    issued_at      DATE NOT NULL DEFAULT CURRENT_DATE,
    paid_at        TIMESTAMPTZ,
    view_token     TEXT NOT NULL UNIQUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_client_id ON invoices(client_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_view_token ON invoices(view_token);

-- +goose Down
DROP TABLE invoices;
DROP TYPE invoice_status;
