-- +goose Up

-- Orgs table (CairnPost-compatible)
CREATE TABLE orgs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Default org for existing data
INSERT INTO orgs (id, name, slug)
VALUES ('00000000-0000-0000-0000-000000000001', 'Firefly Software', 'firefly-software');

-- Users: add UUID, org_id, name, role; rename password → password_hash
ALTER TABLE users ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE users ADD COLUMN org_id UUID REFERENCES orgs(id);
ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'admin';
ALTER TABLE users RENAME COLUMN password TO password_hash;

-- Sessions: add UUID, token_hash
ALTER TABLE sessions ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE sessions ADD COLUMN token_hash TEXT;

-- Clients
ALTER TABLE clients ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE clients ADD COLUMN org_id UUID REFERENCES orgs(id);

-- Invoices
ALTER TABLE invoices ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE invoices ADD COLUMN org_id UUID REFERENCES orgs(id);

-- Invoice line items
ALTER TABLE invoice_line_items ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;

-- Invoice sequences
ALTER TABLE invoice_sequences ADD COLUMN org_id UUID REFERENCES orgs(id);

-- Settings
ALTER TABLE settings ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE settings ADD COLUMN org_id UUID REFERENCES orgs(id);

-- Expense categories
ALTER TABLE expense_categories ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE expense_categories ADD COLUMN org_id UUID REFERENCES orgs(id);

-- Expenses
ALTER TABLE expenses ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE expenses ADD COLUMN org_id UUID REFERENCES orgs(id);

-- +goose Down
ALTER TABLE expenses DROP COLUMN IF EXISTS org_id, DROP COLUMN IF EXISTS uuid;
ALTER TABLE expense_categories DROP COLUMN IF EXISTS org_id, DROP COLUMN IF EXISTS uuid;
ALTER TABLE settings DROP COLUMN IF EXISTS org_id, DROP COLUMN IF EXISTS uuid;
ALTER TABLE invoice_sequences DROP COLUMN IF EXISTS org_id;
ALTER TABLE invoice_line_items DROP COLUMN IF EXISTS uuid;
ALTER TABLE invoices DROP COLUMN IF EXISTS org_id, DROP COLUMN IF EXISTS uuid;
ALTER TABLE clients DROP COLUMN IF EXISTS org_id, DROP COLUMN IF EXISTS uuid;
ALTER TABLE sessions DROP COLUMN IF EXISTS token_hash, DROP COLUMN IF EXISTS uuid;
ALTER TABLE users RENAME COLUMN password_hash TO password;
ALTER TABLE users DROP COLUMN IF EXISTS role, DROP COLUMN IF EXISTS name, DROP COLUMN IF EXISTS org_id, DROP COLUMN IF EXISTS uuid;
DROP TABLE IF EXISTS orgs;
