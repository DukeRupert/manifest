# Manifest — Phase 2: Invoicing

## Overview

Builds the core invoicing system on top of the Phase 1 foundation. Covers the invoice + line items data model, invoice numbering (`INV-{CLIENT_SLUG}-{SEQUENCE}`), status state machine, a settings table for business info and default tax rate, and the public hosted invoice page (token-gated, no auth required).

PDF generation is deferred — the hosted page serves as the primary invoice delivery mechanism in Phase 3.

---

## New Migrations

### 00004_create_settings.sql

```sql
-- +goose Up
CREATE TABLE settings (
    id                BIGSERIAL PRIMARY KEY,
    business_name     TEXT NOT NULL DEFAULT '',
    business_address  TEXT NOT NULL DEFAULT '',
    business_email    TEXT NOT NULL DEFAULT '',
    default_tax_rate  NUMERIC(5,2) NOT NULL DEFAULT 15.00,  -- percentage, e.g. 15.00
    stripe_pk         TEXT NOT NULL DEFAULT '',             -- publishable key (safe to render client-side)
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Single row, always exists
INSERT INTO settings DEFAULT VALUES;

-- +goose Down
DROP TABLE settings;
```

> Always query `SELECT * FROM settings LIMIT 1`. There is exactly one row.

---

### 00005_create_invoices.sql

```sql
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
    number         TEXT NOT NULL UNIQUE,    -- e.g. INV-ACME-0042
    client_id      BIGINT NOT NULL REFERENCES clients(id),
    status         invoice_status NOT NULL DEFAULT 'draft',
    tax_rate       NUMERIC(5,2) NOT NULL,   -- snapshot of rate at time of creation
    notes          TEXT,
    due_date       DATE,
    issued_at      DATE NOT NULL DEFAULT CURRENT_DATE,
    paid_at        TIMESTAMPTZ,
    view_token     TEXT NOT NULL UNIQUE,    -- random token for public URL
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_client_id ON invoices(client_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_view_token ON invoices(view_token);

-- +goose Down
DROP TABLE invoices;
DROP TYPE invoice_status;
```

---

### 00006_create_invoice_line_items.sql

```sql
-- +goose Up
CREATE TABLE invoice_line_items (
    id          BIGSERIAL PRIMARY KEY,
    invoice_id  BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity    NUMERIC(10,2) NOT NULL DEFAULT 1,
    unit_price  NUMERIC(10,2) NOT NULL,           -- in dollars
    position    INT NOT NULL DEFAULT 0,            -- display order
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_line_items_invoice_id ON invoice_line_items(invoice_id);

-- +goose Down
DROP TABLE invoice_line_items;
```

---

### 00007_create_invoice_sequence.sql

```sql
-- +goose Up

-- Per-client invoice sequence counter
CREATE TABLE invoice_sequences (
    client_id   BIGINT PRIMARY KEY REFERENCES clients(id),
    next_val    BIGINT NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE invoice_sequences;
```

---

## Invoice Numbering

Format: `INV-{SLUG}-{SEQUENCE}` where sequence is zero-padded to 4 digits.

Examples:
- `INV-ACME-0001`
- `INV-WSCONTRACTING-0003`
- `INV-1889COFFEE-0012`

### Generation (atomic)

Use a Postgres advisory lock or `FOR UPDATE` to safely increment per-client:

```go
func (s *Store) NextInvoiceNumber(ctx context.Context, tx pgx.Tx, clientID int64, slug string) (string, error) {
    var next int64
    err := tx.QueryRow(ctx, `
        INSERT INTO invoice_sequences (client_id, next_val)
        VALUES ($1, 2)
        ON CONFLICT (client_id) DO UPDATE
            SET next_val = invoice_sequences.next_val + 1
        RETURNING next_val - 1
    `, clientID).Scan(&next)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("INV-%s-%04d", strings.ToUpper(slug), next), nil
}
```

> Always call this inside the same transaction that creates the invoice row.

---

## Invoice Model

```go
package invoice

import (
    "time"
    "manifest/internal/client"
)

type Status string

const (
    StatusDraft  Status = "draft"
    StatusSent   Status = "sent"
    StatusViewed Status = "viewed"
    StatusPaid   Status = "paid"
    StatusVoid   Status = "void"
)

type LineItem struct {
    ID          int64
    InvoiceID   int64
    Description string
    Quantity    float64
    UnitPrice   float64   // dollars
    Position    int
}

func (li LineItem) Subtotal() float64 {
    return li.Quantity * li.UnitPrice
}

type Invoice struct {
    ID         int64
    Number     string
    Client     client.Client
    Status     Status
    TaxRate    float64
    Notes      string
    DueDate    *time.Time
    IssuedAt   time.Time
    PaidAt     *time.Time
    ViewToken  string
    LineItems  []LineItem
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

func (inv *Invoice) Subtotal() float64 {
    var total float64
    for _, li := range inv.LineItems {
        total += li.Subtotal()
    }
    return total
}

func (inv *Invoice) TaxAmount() float64 {
    return inv.Subtotal() * (inv.TaxRate / 100)
}

func (inv *Invoice) Total() float64 {
    return inv.Subtotal() + inv.TaxAmount()
}
```

---

## Status State Machine

Valid transitions only — enforce in the store layer, not just the handler:

```
draft  → sent   (user clicks "Send Invoice")
draft  → void   (user discards draft)
sent   → viewed (public page loaded — auto-transition via middleware)
sent   → paid   (Stripe webhook — Phase 3)
sent   → void   (user voids after sending)
viewed → paid   (Stripe webhook — Phase 3)
viewed → void   (user voids)
paid   → (terminal — no transitions)
void   → (terminal — no transitions)
```

```go
var validTransitions = map[Status][]Status{
    StatusDraft:  {StatusSent, StatusVoid},
    StatusSent:   {StatusViewed, StatusPaid, StatusVoid},
    StatusViewed: {StatusPaid, StatusVoid},
    StatusPaid:   {},
    StatusVoid:   {},
}

func (s *Store) Transition(ctx context.Context, invoiceID int64, to Status) error {
    inv, err := s.Get(ctx, invoiceID)
    if err != nil {
        return err
    }
    allowed := validTransitions[inv.Status]
    for _, a := range allowed {
        if a == to {
            return s.setStatus(ctx, invoiceID, to)
        }
    }
    return fmt.Errorf("invalid transition: %s → %s", inv.Status, to)
}
```

---

## Settings Model

```go
package settings

type Settings struct {
    ID              int64
    BusinessName    string
    BusinessAddress string
    BusinessEmail   string
    DefaultTaxRate  float64
    StripePK        string
}
```

Settings are loaded at startup and cached in memory (or re-fetched per request — simple enough that either works). A single settings page lets the user update all fields.

---

## View Token (Public Invoice URL)

Each invoice has a `view_token` — a 32-byte crypto-random hex string generated at creation time.

Public URL: `https://manifest.fireflysoftware.dev/i/{view_token}`

When this page is loaded:
1. Look up invoice by token (return 404 if not found or voided)
2. If status is `sent`, transition to `viewed`
3. Render the hosted invoice page with client info, line items, totals
4. (Phase 3) Embed Stripe Payment Element

```go
func GenerateViewToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

---

## Routes (Phase 2 additions)

```
# Settings
GET  /settings                    → settings.Show          [authenticated]
POST /settings                    → settings.Update        [authenticated]

# Invoices
GET  /invoices                    → invoice.List           [authenticated]
GET  /invoices/new                → invoice.New            [authenticated]
POST /invoices                    → invoice.Create         [authenticated]
GET  /invoices/{id}               → invoice.Show           [authenticated]
GET  /invoices/{id}/edit          → invoice.Edit           [authenticated]
POST /invoices/{id}               → invoice.Update         [authenticated]
POST /invoices/{id}/send          → invoice.Send           [authenticated]
POST /invoices/{id}/void          → invoice.Void           [authenticated]

# Public (no auth)
GET  /i/{token}                   → invoice.PublicView
```

---

## Invoice Creation Flow

1. User selects a client (must exist)
2. Tax rate pre-filled from `settings.default_tax_rate` — editable per invoice
3. User adds line items (description, qty, unit price)
4. Subtotal / tax / total computed client-side with Alpine.js as items are entered
5. Due date optional
6. On save: wrap in transaction → generate invoice number → insert invoice → insert line items → commit
7. Invoice created in `draft` status
8. User reviews, then clicks "Send" → status transitions to `sent` → (Phase 3: email with link)

---

## Invoice List — Useful Queries

```sql
-- All invoices with client name and totals
SELECT
    i.id,
    i.number,
    c.name AS client_name,
    i.status,
    i.due_date,
    i.issued_at,
    COALESCE(SUM(li.quantity * li.unit_price), 0) AS subtotal,
    COALESCE(SUM(li.quantity * li.unit_price), 0) * (i.tax_rate / 100) AS tax,
    COALESCE(SUM(li.quantity * li.unit_price), 0) * (1 + i.tax_rate / 100) AS total
FROM invoices i
JOIN clients c ON c.id = i.client_id
LEFT JOIN invoice_line_items li ON li.invoice_id = i.id
GROUP BY i.id, c.name
ORDER BY i.created_at DESC;
```

---

## Phase 2 Checklist

- [ ] Run migrations 00004–00007
- [ ] Implement `settings` package (store, handler)
- [ ] Implement `invoice` package (model, store, handlers)
- [ ] Invoice number generation (atomic, per-client sequence)
- [ ] View token generation
- [ ] Status transition enforcement
- [ ] Public invoice page route (`/i/{token}`) — renders invoice, marks viewed
- [ ] Invoice list with status badges and totals
- [ ] Create/edit invoice with line items (Alpine.js running totals)
- [ ] Send / void actions
- [ ] Settings page (business info + default tax rate)
- [ ] Test: create invoice, send, load public URL, verify viewed transition

---

## What's Not In This Phase

- Stripe Payment Element on public page (Phase 3)
- Email delivery of invoice link (Phase 3 — send the link manually for now)
- PDF export
- Expenses (Phase 4)
- Reporting (Phase 5)
