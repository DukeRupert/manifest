# Manifest — Multi-Tenancy Migration Plan

## Goal

Migrate Manifest from single-tenant (1 user, no org, BIGSERIAL PKs) to multi-tenant (orgs, UUID identifiers, org-scoped data, hashed session tokens) — compatible with CairnPost's auth model so the two products can integrate down the road.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Service layer?** | No — keep 2-layer (Store → Handler) | Migration is already a large change surface. Add a service layer as a separate refactor later if needed. |
| **PK strategy** | Keep BIGSERIAL as internal PK, add UUID as application-facing ID | Avoids touching every FK. Internal joins use BIGSERIAL; handlers/URLs use UUID. |
| **Session tokens** | Hash with SHA-256, store hash in DB | Aligns with CairnPost. Raw token stays in cookie only. |
| **Org resolution** | From auth context (middleware extracts org_id from session → user → org) | Every store method reads `auth.OrgID(ctx)`. No per-request store construction needed. |
| **Column rename** | `users.password` → `users.password_hash` | CairnPost compatibility. |

---

## Phase 0: Schema Preparation (No Downtime)

Run while the existing app is live — all changes are additive.

### Migration 00011: Create orgs table and add shadow columns

```sql
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

-- Shadow columns on all tables
ALTER TABLE users ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE users ADD COLUMN org_id UUID REFERENCES orgs(id);
ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'admin';
ALTER TABLE users RENAME COLUMN password TO password_hash;

ALTER TABLE sessions ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE sessions ADD COLUMN token_hash TEXT;

ALTER TABLE clients ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE clients ADD COLUMN org_id UUID REFERENCES orgs(id);

ALTER TABLE invoices ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE invoices ADD COLUMN org_id UUID REFERENCES orgs(id);

ALTER TABLE invoice_line_items ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;

ALTER TABLE invoice_sequences ADD COLUMN org_id UUID REFERENCES orgs(id);

ALTER TABLE settings ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE settings ADD COLUMN org_id UUID REFERENCES orgs(id);

ALTER TABLE expense_categories ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE expense_categories ADD COLUMN org_id UUID REFERENCES orgs(id);

ALTER TABLE expenses ADD COLUMN uuid UUID DEFAULT gen_random_uuid() UNIQUE;
ALTER TABLE expenses ADD COLUMN org_id UUID REFERENCES orgs(id);

-- +goose Down
ALTER TABLE expenses DROP COLUMN org_id, DROP COLUMN uuid;
ALTER TABLE expense_categories DROP COLUMN org_id, DROP COLUMN uuid;
ALTER TABLE settings DROP COLUMN org_id, DROP COLUMN uuid;
ALTER TABLE invoice_sequences DROP COLUMN org_id;
ALTER TABLE invoice_line_items DROP COLUMN uuid;
ALTER TABLE invoices DROP COLUMN org_id, DROP COLUMN uuid;
ALTER TABLE clients DROP COLUMN org_id, DROP COLUMN uuid;
ALTER TABLE sessions DROP COLUMN token_hash, DROP COLUMN uuid;
ALTER TABLE users RENAME COLUMN password_hash TO password;
ALTER TABLE users DROP COLUMN role, DROP COLUMN name, DROP COLUMN org_id, DROP COLUMN uuid;
DROP TABLE orgs;
```

### Migration 00012: Backfill existing data

```sql
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
-- Current: id column stores raw 64-char hex token
-- New: token_hash = hex(sha256(hex_token_as_bytes))
UPDATE sessions SET token_hash = encode(sha256(id::bytea), 'hex') WHERE token_hash IS NULL;

-- +goose Down
-- No undo needed; data remains valid with old columns
```

### Migration 00013: Enforce constraints (requires maintenance window)

```sql
-- +goose Up

-- Make org_id NOT NULL
ALTER TABLE users ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE clients ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE invoices ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE invoice_sequences ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE settings ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE expense_categories ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE expenses ALTER COLUMN org_id SET NOT NULL;

-- Org-scoped unique constraints (replace global uniques)
ALTER TABLE users DROP CONSTRAINT users_email_key;
ALTER TABLE users ADD CONSTRAINT users_org_email_unique UNIQUE (org_id, email);

DROP INDEX IF EXISTS idx_clients_slug;
ALTER TABLE clients DROP CONSTRAINT clients_slug_key;
ALTER TABLE clients ADD CONSTRAINT clients_org_slug_unique UNIQUE (org_id, slug);

ALTER TABLE invoices DROP CONSTRAINT invoices_number_key;
ALTER TABLE invoices ADD CONSTRAINT invoices_org_number_unique UNIQUE (org_id, number);

ALTER TABLE expense_categories DROP CONSTRAINT expense_categories_name_key;
ALTER TABLE expense_categories ADD CONSTRAINT expense_categories_org_name_unique UNIQUE (org_id, name);

-- invoice_sequences: composite PK (org_id, client_id)
ALTER TABLE invoice_sequences DROP CONSTRAINT invoice_sequences_pkey;
ALTER TABLE invoice_sequences ADD PRIMARY KEY (org_id, client_id);

-- Session token hash
ALTER TABLE sessions ALTER COLUMN token_hash SET NOT NULL;
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(token_hash);

-- Performance indexes
CREATE INDEX idx_clients_org_id ON clients(org_id);
CREATE INDEX idx_invoices_org_id ON invoices(org_id);
CREATE INDEX idx_expenses_org_id ON expenses(org_id);
CREATE INDEX idx_expense_categories_org_id ON expense_categories(org_id);

-- +goose Down
-- (reverse operations — omitted for brevity)
```

---

## Phase 1: Auth Overhaul

Deploy together with migration 00013.

### `internal/auth/session.go`

- `Session` struct: `UserID` → `string` (UUID), add `OrgID string`
- `GenerateSessionID()` unchanged (returns random hex token)
- Add `HashToken(raw string) string` helper — `hex(sha256(raw_bytes))`
- `CreateSession(ctx, userID string)`: insert with `gen_random_uuid()` PK, store `token_hash`
- `GetSession(ctx, rawToken string)`: hash token, `SELECT ... JOIN users ON ... WHERE token_hash = $1`
- `DeleteSession(ctx, rawToken string)`: hash token, `DELETE WHERE token_hash = $1`

### `internal/auth/middleware.go`

- Context carries both `userID` and `orgID` (both UUID strings)
- Add `OrgID(ctx) string` helper alongside existing `UserID(ctx)`
- `UserID()` return type changes from `int64` to `string`

### `internal/auth/handler.go`

- Login query: `SELECT uuid, password_hash FROM users WHERE email = $1`
- `CreateSession()` receives UUID string

### `cmd/manifest/seed.go`

- Prompt for org name (default: "Firefly Software")
- Find-or-create org by slug
- Insert user with `org_id`, `password_hash`, `name`, `role`

---

## Phase 2: Org-Scoped Stores

Every store method reads `orgID := auth.OrgID(ctx)` and adds org_id to queries.

### Pattern (applies to all stores)

```go
// Before
func (s *Store) List(ctx context.Context) ([]Client, error) {
    rows, err := s.pool.Query(ctx,
        `SELECT ... FROM clients WHERE archived_at IS NULL ORDER BY name`)
}

// After
func (s *Store) List(ctx context.Context) ([]Client, error) {
    orgID := auth.OrgID(ctx)
    rows, err := s.pool.Query(ctx,
        `SELECT ... FROM clients WHERE archived_at IS NULL AND org_id = $1 ORDER BY name`, orgID)
}
```

### ID strategy in stores

- Go structs: `ID string` mapped to the `uuid` column
- Handlers parse `r.PathValue("id")` as a string (UUID), no `strconv.ParseInt`
- Store queries: `WHERE uuid = $1 AND org_id = $2`
- Internal FKs (e.g., `invoices.client_id BIGINT`) stay as BIGSERIAL references
- When creating an invoice, look up client's BIGSERIAL `id` from its UUID for the FK insert

### Files changed

| File | Changes |
|------|---------|
| `internal/client/model.go` | `ID` → `string`, add `OrgID string` |
| `internal/client/store.go` | Org-scoped queries, UUID-based lookups |
| `internal/client/handler.go` | Remove `strconv.ParseInt`, UUID path params |
| `internal/invoice/model.go` | All IDs → `string`, add `OrgID string` |
| `internal/invoice/store.go` | Org-scoped queries, composite sequence key `(org_id, client_id)`, client UUID→BIGSERIAL lookup |
| `internal/invoice/handler.go` | UUID path params and form values |
| `internal/expense/model.go` | All IDs → `string`, add `OrgID` |
| `internal/expense/store.go` | Org-scoped queries |
| `internal/expense/handler.go` | UUID path params |
| `internal/settings/store.go` | `WHERE org_id = $1` on Get/Update |
| `internal/reports/store.go` | Org-scoped all queries |
| `internal/payment/payment.go` | UUID in `CreateIntentParams` |
| `internal/payment/webhook.go` | Invoice ID type change (minor) |

### Special cases

- **`PublicView` / `PaymentConfirmed`** (`/i/{token}`): Token-based, no auth context. Globally unique tokens — no org scoping needed.
- **Webhook handler**: Called by Stripe, no auth context. `GetByPaymentIntent()` stays global (payment_intent_id is globally unique).
- **`invoice_sequences`**: Composite key `(org_id, client_id)`. `NextInvoiceNumber()` uses org_id from context.

---

## Phase 3: Templates and Frontend

### `templates/models.go`

All `int64` ID fields → `string`:
- `ClientView.ID`
- `InvoiceListItemView.ID`
- `LineItemView.ID`
- `InvoiceView.ID`, `InvoiceView.ClientID`
- `CategoryView.ID`
- `ExpenseView.ID`, `ExpenseView.CategoryID`
- `ExpenseListData.FilterCat`

### All `.templ` files

- URL format strings: `fmt.Sprintf("/clients/%d", ...)` → `fmt.Sprintf("/clients/%s", ...)`
- Same for invoices, expenses, categories
- Form hidden fields: integer formatting → string value directly

---

## Phase 4: PK Cleanup (Deferred — Optional)

Once multi-tenancy is stable, optionally:
1. Add UUID FK columns alongside BIGSERIAL FK columns
2. Backfill UUID FKs
3. Drop BIGSERIAL FK columns
4. Make UUID the PK, rename `uuid` → `id`

This is not required for correctness — the dual-column approach works fine indefinitely.

---

## Phase 5: Org Management (Future Feature)

- Org creation (CLI or UI)
- User invitation flow
- Org-level settings
- Role-based access (admin vs member)

---

## Deployment Sequence

| Step | Action | Downtime? |
|------|--------|-----------|
| 1 | Run migration 00011 (add columns) | No |
| 2 | Run migration 00012 (backfill) | No |
| 3 | **Start maintenance window** | Yes |
| 4 | Run migration 00013 (constraints) | Yes |
| 5 | Deploy new code (Phases 1–3) | Yes |
| 6 | Verify: login, list clients, create invoice, public page | Yes |
| 7 | **End maintenance window** | |

Estimated window: 15–30 minutes.

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Stripe webhooks during downtime | Stripe retries for 72 hours — no payment data lost |
| Public invoice links (`/i/{token}`) | Token-based, globally unique — unaffected |
| Session invalidation on deploy | Migration 00012 pre-hashes tokens; new code hashes cookie value the same way |
| Invoice numbering | Composite key `(org_id, client_id)` preserves existing sequences |
| Rollback | Migrations 00011-00012 are additive; if 00013 + code deploy fails, restore from backup (small dataset) |
