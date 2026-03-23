# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Manifest is a single-tenant invoicing and billing web app for Firefly Software. It handles clients, invoices with per-client sequential numbering, Stripe payments (card + ACH), manual expense tracking, and basic financial reporting (Revenue, AR, P&L).

## Tech Stack

- **Language:** Go 1.23
- **Database:** PostgreSQL 16 (via `pgx/v5`)
- **Migrations:** goose (SQL files in `migrations/`)
- **Templating:** templ (`.templ` files)
- **Frontend interactivity:** Alpine.js, htmx
- **Payments:** Stripe Payment Element (`stripe-go/v84`, vanilla Stripe.js — no npm)
- **Auth:** Session-based (bcrypt passwords, 32-byte hex session tokens, `manifest_session` cookie)
- **Deployment:** Docker Compose → Caddy reverse proxy, app on port 8080 internally

## Build & Run

```bash
go build -o manifest ./cmd/manifest     # build
./manifest                               # run server
./manifest seed                          # create/reset admin user interactively
docker compose up                        # full stack with Postgres
```

## Migrations

```bash
goose -dir migrations postgres "$DATABASE_URL" up
goose -dir migrations postgres "$DATABASE_URL" down
```

## Architecture

Standard Go project layout with domain packages under `internal/`:

- `cmd/manifest/` — Entry point with subcommand dispatch (`seed` or server)
- `internal/auth/` — Session store, login/logout handlers, auth middleware
- `internal/client/` — Client CRUD (model, store, handlers)
- `internal/invoice/` — Invoice lifecycle: CRUD, line items, status state machine, per-client sequence numbering, view tokens for public pages
- `internal/payment/` — Stripe PaymentIntent creation, webhook handler
- `internal/expense/` — Expense and category CRUD
- `internal/settings/` — Single-row settings table (business info, default tax rate, Stripe publishable key)
- `internal/db/` — Connection pool initialization
- `internal/server/` — HTTP server setup, route registration
- `templates/` — templ UI files
- `static/` — CSS, JS assets

Each domain package follows the pattern: `model.go` (structs), `store.go` (DB queries), `handler.go` (HTTP handlers).

## Key Design Decisions

- **Single-tenant, single admin user** — no registration flow, user seeded via CLI
- **Invoice numbering:** `INV-{CLIENT_SLUG}-{SEQUENCE}` (e.g., `INV-ACME-0042`), generated atomically via `invoice_sequences` table with `FOR UPDATE`/upsert inside the invoice creation transaction
- **Invoice status state machine:** draft → sent → viewed → paid (terminal) / void (terminal). Transitions enforced in the store layer. `viewed` is auto-set when public page is loaded
- **Public invoice page:** `/i/{view_token}` — token-gated, no auth. Embeds Stripe Payment Element for payment
- **POST for mutations** — all create/update/delete/archive use POST (htmx-friendly, no JS method override)
- **Soft delete:** Clients use `archived_at`; invoices use `void` status
- **Client slugs** are immutable after any invoice exists for that client
- **Settings** is always a single row (`SELECT * FROM settings LIMIT 1`)
- **Amounts:** Stored as `NUMERIC(10,2)` in dollars in DB; converted to cents server-side for Stripe

## Environment Variables

```
APP_ENV, APP_SECRET, DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, PORT
STRIPE_SECRET_KEY, STRIPE_WEBHOOK_SECRET  # server-side only
```

Stripe publishable key is stored in the `settings` table, not env vars.

## Phase Specs

Detailed implementation specs are in the repo root:
- `manifest-phase-1-foundation.md` — Project scaffold, auth, clients
- `manifest-phase-2-invoicing.md` — Invoices, line items, settings, public invoice page
- `manifest-phase-3-payments.md` — Stripe integration, webhooks
- `manifest-phase-4-5-expenses-reporting.md` — Expenses, categories, financial reports
