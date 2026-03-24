-- +goose Up

-- Make org_id NOT NULL on all tables
ALTER TABLE users ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE clients ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE invoices ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE invoice_sequences ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE settings ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE expense_categories ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE expenses ALTER COLUMN org_id SET NOT NULL;

-- Org-scoped unique constraints (replace global uniques)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE users ADD CONSTRAINT users_org_email_unique UNIQUE (org_id, email);

DROP INDEX IF EXISTS idx_clients_slug;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_slug_key;
ALTER TABLE clients ADD CONSTRAINT clients_org_slug_unique UNIQUE (org_id, slug);

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_number_key;
ALTER TABLE invoices ADD CONSTRAINT invoices_org_number_unique UNIQUE (org_id, number);

ALTER TABLE expense_categories DROP CONSTRAINT IF EXISTS expense_categories_name_key;
ALTER TABLE expense_categories ADD CONSTRAINT expense_categories_org_name_unique UNIQUE (org_id, name);

-- Invoice sequences: composite PK (org_id, client_id)
ALTER TABLE invoice_sequences DROP CONSTRAINT IF EXISTS invoice_sequences_pkey;
ALTER TABLE invoice_sequences ADD PRIMARY KEY (org_id, client_id);

-- Session token hash: NOT NULL + unique index
ALTER TABLE sessions ALTER COLUMN token_hash SET NOT NULL;
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(token_hash);

-- Performance indexes for org-scoped queries
CREATE INDEX idx_clients_org_id ON clients(org_id);
CREATE INDEX idx_invoices_org_id ON invoices(org_id);
CREATE INDEX idx_expenses_org_id ON expenses(org_id);
CREATE INDEX idx_expense_categories_org_id ON expense_categories(org_id);
CREATE INDEX idx_users_org_id ON users(org_id);

-- +goose Down
DROP INDEX IF EXISTS idx_users_org_id;
DROP INDEX IF EXISTS idx_expense_categories_org_id;
DROP INDEX IF EXISTS idx_expenses_org_id;
DROP INDEX IF EXISTS idx_invoices_org_id;
DROP INDEX IF EXISTS idx_clients_org_id;
DROP INDEX IF EXISTS idx_sessions_token_hash;
ALTER TABLE sessions ALTER COLUMN token_hash DROP NOT NULL;

ALTER TABLE invoice_sequences DROP CONSTRAINT IF EXISTS invoice_sequences_pkey;
ALTER TABLE invoice_sequences ADD PRIMARY KEY (client_id);

ALTER TABLE expense_categories DROP CONSTRAINT IF EXISTS expense_categories_org_name_unique;
ALTER TABLE expense_categories ADD CONSTRAINT expense_categories_name_key UNIQUE (name);

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_org_number_unique;
ALTER TABLE invoices ADD CONSTRAINT invoices_number_key UNIQUE (number);

ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_org_slug_unique;
ALTER TABLE clients ADD CONSTRAINT clients_slug_key UNIQUE (slug);
CREATE INDEX idx_clients_slug ON clients(slug);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_org_email_unique;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE expenses ALTER COLUMN org_id DROP NOT NULL;
ALTER TABLE expense_categories ALTER COLUMN org_id DROP NOT NULL;
ALTER TABLE settings ALTER COLUMN org_id DROP NOT NULL;
ALTER TABLE invoice_sequences ALTER COLUMN org_id DROP NOT NULL;
ALTER TABLE invoices ALTER COLUMN org_id DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN org_id DROP NOT NULL;
ALTER TABLE users ALTER COLUMN org_id DROP NOT NULL;
