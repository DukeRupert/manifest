# Manifest — Phase 4: Expenses

## Overview

Manual expense entry. No receipt upload, no bank import — just a clean form to log what was spent, when, who it was paid to, and which category it falls under. Categories are user-managed.

---

## Migrations

### 00009_create_expense_categories.sql

```sql
-- +goose Up
CREATE TABLE expense_categories (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed sensible defaults
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
```

---

### 00010_create_expenses.sql

```sql
-- +goose Up
CREATE TABLE expenses (
    id          BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES expense_categories(id),
    vendor      TEXT NOT NULL,
    amount      NUMERIC(10,2) NOT NULL,   -- dollars
    notes       TEXT,
    date        DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expenses_category_id ON expenses(category_id);
CREATE INDEX idx_expenses_date ON expenses(date);

-- +goose Down
DROP TABLE expenses;
```

---

## Expense Model

```go
package expense

import "time"

type Category struct {
    ID   int64
    Name string
}

type Expense struct {
    ID         int64
    Category   Category
    Vendor     string
    Amount     float64
    Notes      string
    Date       time.Time
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

---

## Routes

```
# Categories
GET  /expenses/categories         → category.List      [authenticated]
GET  /expenses/categories/new     → category.New       [authenticated]
POST /expenses/categories         → category.Create    [authenticated]
POST /expenses/categories/{id}    → category.Update    [authenticated]
POST /expenses/categories/{id}/delete → category.Delete [authenticated]

# Expenses
GET  /expenses                    → expense.List       [authenticated]
GET  /expenses/new                → expense.New        [authenticated]
POST /expenses                    → expense.Create     [authenticated]
GET  /expenses/{id}/edit          → expense.Edit       [authenticated]
POST /expenses/{id}               → expense.Update     [authenticated]
POST /expenses/{id}/delete        → expense.Delete     [authenticated]
```

---

## Expense List — Useful Queries

```sql
-- Expenses with category name, newest first
SELECT
    e.id,
    e.date,
    e.vendor,
    ec.name AS category,
    e.amount,
    e.notes
FROM expenses e
JOIN expense_categories ec ON ec.id = e.category_id
ORDER BY e.date DESC;

-- Monthly expense total by category (used in reporting)
SELECT
    ec.name AS category,
    SUM(e.amount) AS total
FROM expenses e
JOIN expense_categories ec ON ec.id = e.category_id
WHERE DATE_TRUNC('month', e.date) = DATE_TRUNC('month', $1::date)
GROUP BY ec.name
ORDER BY total DESC;
```

---

## Phase 4 Checklist

- [ ] Run migrations 00009–00010
- [ ] Implement `expense` package (model, store, handlers)
- [ ] Category CRUD (including delete guard — don't delete if expenses reference it)
- [ ] Expense list with date range filter and category filter
- [ ] Create/edit expense form
- [ ] Test: create category, log expense, filter by date range

---

---

# Manifest — Phase 5: Reporting

## Overview

Basic financial reporting derived entirely from existing data — no new tables required. Three views: Revenue, Outstanding AR, and P&L.

---

## Report Definitions

### 1. Revenue Report

What came in, by period.

```sql
-- Revenue by month for a given year
SELECT
    DATE_TRUNC('month', paid_at) AS month,
    COUNT(*) AS invoice_count,
    SUM(amount_paid_cents) / 100.0 AS revenue
FROM invoices
WHERE status = 'paid'
  AND EXTRACT(YEAR FROM paid_at) = $1
GROUP BY month
ORDER BY month;
```

Display: bar chart (Alpine.js + simple SVG or a lightweight chart lib) plus a table breakdown. Show current month prominently, YTD total in the header.

---

### 2. Outstanding AR

Unpaid invoices, sorted by most overdue first.

```sql
SELECT
    i.number,
    c.name AS client,
    i.due_date,
    i.issued_at,
    CURRENT_DATE - i.due_date AS days_overdue,
    COALESCE(SUM(li.quantity * li.unit_price), 0) * (1 + i.tax_rate / 100) AS total
FROM invoices i
JOIN clients c ON c.id = i.client_id
LEFT JOIN invoice_line_items li ON li.invoice_id = i.id
WHERE i.status IN ('sent', 'viewed')
GROUP BY i.id, c.name
ORDER BY i.due_date ASC NULLS LAST;
```

Highlight invoices where `due_date < CURRENT_DATE` as overdue.

---

### 3. P&L (Profit & Loss)

Revenue minus expenses, by period.

```sql
-- Revenue side
SELECT
    DATE_TRUNC('month', paid_at) AS month,
    SUM(amount_paid_cents) / 100.0 AS revenue
FROM invoices
WHERE status = 'paid'
  AND paid_at >= $1 AND paid_at < $2
GROUP BY month;

-- Expense side
SELECT
    DATE_TRUNC('month', date) AS month,
    SUM(amount) AS expenses
FROM expenses
WHERE date >= $1 AND date < $2
GROUP BY month;
```

Join these in Go (not SQL) to produce a per-month P&L table:

| Month | Revenue | Expenses | Net |
|---|---|---|---|
| Jan 2026 | $8,400 | $1,230 | $7,170 |
| Feb 2026 | $6,200 | $980 | $5,220 |

Show a YTD summary row at the bottom.

---

## Report Filters

All reports support a date range picker. Defaults:
- Revenue: current year
- AR: all outstanding (no date filter)
- P&L: current year

Date range passed as query params: `?from=2026-01-01&to=2026-12-31`

---

## Routes

```
GET  /reports                     → reports.Index      [authenticated]  (summary dashboard)
GET  /reports/revenue             → reports.Revenue    [authenticated]
GET  /reports/ar                  → reports.AR         [authenticated]
GET  /reports/pl                  → reports.PL         [authenticated]
```

---

## Reports Index (Dashboard)

The `/reports` page (also a reasonable candidate for `/` home) shows:

- **This month revenue** — sum of paid invoices this calendar month
- **Outstanding AR** — count and total of unpaid invoices
- **YTD revenue** — sum of paid invoices this year
- **YTD expenses** — sum of all expenses this year
- **YTD net** — revenue minus expenses

These are cheap queries and safe to run on every page load.

---

## Phase 5 Checklist

- [ ] Implement `reports` package (store queries, handlers)
- [ ] Revenue report — monthly breakdown + YTD
- [ ] AR report — outstanding invoices, overdue highlighted
- [ ] P&L report — revenue vs expenses by month, net column
- [ ] Date range filter on all reports
- [ ] Reports index / dashboard summary cards
- [ ] Test: verify totals match raw invoice and expense data
