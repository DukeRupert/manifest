-- +goose Up
CREATE TABLE invoice_sequences (
    client_id   BIGINT PRIMARY KEY REFERENCES clients(id),
    next_val    BIGINT NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE invoice_sequences;
