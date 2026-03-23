-- +goose Up
CREATE TABLE expenses (
    id          BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES expense_categories(id),
    vendor      TEXT NOT NULL,
    amount      NUMERIC(10,2) NOT NULL,
    notes       TEXT,
    date        DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expenses_category_id ON expenses(category_id);
CREATE INDEX idx_expenses_date ON expenses(date);

-- +goose Down
DROP TABLE expenses;
