-- +goose Up

-- Assign all existing rows to the default org
UPDATE users SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;
UPDATE clients SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;
UPDATE invoices SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;
UPDATE invoice_sequences SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;
UPDATE settings SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;
UPDATE expense_categories SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;
UPDATE expenses SET org_id = '00000000-0000-0000-0000-000000000001' WHERE org_id IS NULL;

-- Hash existing session tokens
-- Current: id column stores raw 64-char hex token string
-- New: token_hash = hex(sha256(hex_string_as_bytes))
UPDATE sessions SET token_hash = encode(sha256(id::bytea), 'hex') WHERE token_hash IS NULL;

-- +goose Down
-- No undo needed; data remains valid with old columns
