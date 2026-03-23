-- +goose Up
CREATE TABLE expense_categories (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO expense_categories (name) VALUES
    ('Software & Subscriptions'),
    ('Hosting & Infrastructure'),
    ('Contractors & Subcontractors'),
    ('Office & Supplies'),
    ('Marketing & Advertising'),
    ('Professional Development'),
    ('Travel'),
    ('Meals & Entertainment'),
    ('Uncategorized');

-- +goose Down
DROP TABLE expense_categories;
