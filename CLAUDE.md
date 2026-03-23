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

## Design Context

### Users
Single admin user (the owner of Firefly Software) managing day-to-day invoicing, client relationships, expense tracking, and financial reporting. Uses the app as a personal command center for business operations. Clients interact only with the public invoice page (`/i/{token}`) to view and pay invoices.

### Brand Personality
**Characterful, capable, lived-in.** The interface has personality — it's not a sterile SaaS dashboard. It feels like a well-worn ship console: warm, familiar, and distinctly yours. The Firefly/sci-fi identity isn't decoration; it's the brand. Clients see it too.

Emotional goals: Using the app should feel **fun and satisfying** — like piloting your own ship. Data is presented with confidence, actions feel decisive, and the theme makes routine billing work feel a little less mundane.

### Aesthetic Direction
**Retro-futuristic / analog-meets-digital.** Inspired by Cowboy Bebop and Firefly — worn textures, warm amber light against deep space blacks, CRT and terminal aesthetics mixed with readable serif typography.

- **Theme:** Dark mode only. Warm foreground (rust, amber, bone) against cool void backgrounds (slate, space).
- **Typography:** Serif-forward (Playfair Display headings, Crimson Pro body) with monospace accents (Share Tech Mono) for labels, badges, and system elements.
- **Shadows:** Glow-only — no drop shadows. Colored glow rings for focus states.
- **Surfaces:** Flat with subtle borders. Card accents via colored left-border strips.
- **Public invoice page:** Full Firefly theme — clients experience the brand identity, not a generic payment form.
- **Anti-references:** Generic SaaS (white/gray/blue), Material Design defaults, anything that looks like every other invoicing app.

### Design Principles

1. **Character over convention** — Lean into the Firefly/sci-fi theme everywhere. Empty states, loading indicators, labels, and micro-copy should have personality. Monospace labels like `CORTEX`, `CREW`, `GUILD` over generic terms when thematically appropriate.
2. **Clarity through hierarchy** — Despite the themed aesthetic, information must be instantly scannable. Use the type scale, color semantics, and spacing intentionally. Numbers (amounts, dates, statuses) are the most important data — make them unmissable.
3. **Warm, not cold** — The dark theme should feel like a ship interior, not a void. Use rust/amber warmth generously. Bone text on space backgrounds. Avoid pure black-on-white or cold blue-gray palettes.
4. **Functional minimalism** — Every element earns its space. No decorative clutter. The theme provides visual interest; the layout stays clean and purposeful. Components from `firefly.tailwind.config.js` (btn, card-vessel, badge, alert, input-cortex, section-label) are the vocabulary.
5. **Accessible by default** — WCAG AA contrast ratios, keyboard navigable, screen reader friendly. The themed aesthetic must not compromise usability.
