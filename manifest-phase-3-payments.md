# Manifest — Phase 3: Payments

## Overview

Wires Stripe into the hosted invoice page. Clients land on `/i/{token}`, see the invoice, and pay via credit/debit card or ACH bank transfer using Stripe's Payment Element. A webhook handler receives Stripe events and marks invoices paid.

No new migrations are needed beyond adding a few columns to `invoices`.

---

## Migration

### 00008_invoices_stripe_columns.sql

```sql
-- +goose Up
ALTER TABLE invoices
    ADD COLUMN stripe_payment_intent_id TEXT,
    ADD COLUMN stripe_charge_id         TEXT,
    ADD COLUMN amount_paid_cents        BIGINT;   -- actual amount paid, in cents

CREATE UNIQUE INDEX idx_invoices_payment_intent
    ON invoices(stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;

-- +goose Down
ALTER TABLE invoices
    DROP COLUMN stripe_payment_intent_id,
    DROP COLUMN stripe_charge_id,
    DROP COLUMN amount_paid_cents;
```

---

## Stripe Configuration

Add to `.env`:

```bash
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
# STRIPE_PK is already stored in settings table (rendered client-side)
```

Stripe secret key and webhook secret are server-side only and never rendered to the client.
The publishable key is read from the `settings` table and embedded in the public invoice page template.

---

## Payment Flow

```
Client loads /i/{token}
    → Server fetches invoice
    → Server creates (or retrieves existing) PaymentIntent via Stripe API
    → Server renders page with:
        - Invoice details
        - client_secret from PaymentIntent
        - Stripe publishable key
    → Client-side: Stripe.js mounts Payment Element using client_secret
    → Client completes payment
    → Stripe confirms payment
    → Stripe fires webhook → POST /webhooks/stripe
    → Webhook handler marks invoice paid
```

---

## PaymentIntent Creation

Create the PaymentIntent server-side when the public invoice page is loaded. Store the intent ID on the invoice so repeat visits reuse the same intent.

```go
package payment

import (
    "github.com/stripe/stripe-go/v84"
    "github.com/stripe/stripe-go/v84/paymentintent"
)

type CreateIntentParams struct {
    InvoiceID     int64
    InvoiceNumber string
    AmountCents   int64     // total in cents
    ClientEmail   string
}

func CreateOrGetIntent(ctx context.Context, store *invoice.Store, params CreateIntentParams) (string, error) {
    // If invoice already has a payment intent, return its client secret
    inv, err := store.Get(ctx, params.InvoiceID)
    if err != nil {
        return "", err
    }
    if inv.StripePaymentIntentID != "" {
        pi, err := paymentintent.Get(inv.StripePaymentIntentID, nil)
        if err != nil {
            return "", err
        }
        return pi.ClientSecret, nil
    }

    // Create new PaymentIntent
    pi, err := paymentintent.New(&stripe.PaymentIntentParams{
        Amount:   stripe.Int64(params.AmountCents),
        Currency: stripe.String("usd"),
        PaymentMethodTypes: stripe.StringSlice([]string{
            "card",
            "us_bank_account",   // ACH
        }),
        ReceiptEmail: stripe.String(params.ClientEmail),
        Metadata: map[string]string{
            "invoice_id":     fmt.Sprintf("%d", params.InvoiceID),
            "invoice_number": params.InvoiceNumber,
        },
    })
    if err != nil {
        return "", err
    }

    // Persist intent ID to invoice
    if err := store.SetPaymentIntent(ctx, params.InvoiceID, pi.ID); err != nil {
        return "", err
    }

    return pi.ClientSecret, nil
}
```

### Amount Calculation

Always calculate total in cents server-side — never trust client-submitted amounts:

```go
func TotalCents(inv *invoice.Invoice) int64 {
    total := inv.Total()                    // float64 dollars
    return int64(math.Round(total * 100))
}
```

---

## Public Invoice Page Handler (updated)

```go
func (h *Handler) PublicView(w http.ResponseWriter, r *http.Request) {
    token := r.PathValue("token")

    inv, err := h.invoiceStore.GetByToken(r.Context(), token)
    if err != nil || inv == nil || inv.Status == invoice.StatusVoid {
        http.NotFound(w, r)
        return
    }

    // Auto-transition sent → viewed
    if inv.Status == invoice.StatusSent {
        _ = h.invoiceStore.Transition(r.Context(), inv.ID, invoice.StatusViewed)
        inv.Status = invoice.StatusViewed
    }

    // If already paid, render paid confirmation page
    if inv.Status == invoice.StatusPaid {
        renderPaidPage(w, r, inv)
        return
    }

    // Create or retrieve PaymentIntent
    clientSecret, err := payment.CreateOrGetIntent(r.Context(), h.invoiceStore, payment.CreateIntentParams{
        InvoiceID:     inv.ID,
        InvoiceNumber: inv.Number,
        AmountCents:   payment.TotalCents(inv),
        ClientEmail:   inv.Client.Email,
    })
    if err != nil {
        http.Error(w, "payment setup failed", http.StatusInternalServerError)
        return
    }

    settings, _ := h.settingsStore.Get(r.Context())

    renderInvoicePage(w, r, inv, clientSecret, settings.StripePK)
}
```

---

## Client-Side Payment Element

Rendered inside the public invoice page template. Uses vanilla Stripe.js — no npm required.

```html
<script src="https://js.stripe.com/v3/"></script>
<script>
  const stripe = Stripe('{{ .StripePK }}');
  const elements = stripe.elements({ clientSecret: '{{ .ClientSecret }}' });

  const paymentElement = elements.create('payment');
  paymentElement.mount('#payment-element');

  document.getElementById('pay-button').addEventListener('click', async () => {
    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: '{{ .ReturnURL }}',  // e.g. /i/{token}/confirmed
      },
    });
    if (error) {
      document.getElementById('error-message').textContent = error.message;
    }
  });
</script>
```

Return URL after payment: `https://manifest.fireflysoftware.dev/i/{token}/confirmed`

This page simply shows a thank-you message. The invoice is marked paid by the webhook, not by the return URL (return URL can arrive before webhook fires).

---

## Webhook Handler

```go
func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
    const maxBodyBytes = int64(65536)
    r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

    payload, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "read error", http.StatusBadRequest)
        return
    }

    sig := r.Header.Get("Stripe-Signature")
    event, err := webhook.ConstructEvent(payload, sig, h.webhookSecret)
    if err != nil {
        http.Error(w, "signature verification failed", http.StatusBadRequest)
        return
    }

    switch event.Type {
    case "payment_intent.succeeded":
        var pi stripe.PaymentIntent
        if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
            http.Error(w, "parse error", http.StatusBadRequest)
            return
        }
        h.handlePaymentSucceeded(r.Context(), &pi)

    case "payment_intent.payment_failed":
        // Log, but no state change needed
    }

    w.WriteHeader(http.StatusOK)
}

func (h *Handler) handlePaymentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) {
    inv, err := h.invoiceStore.GetByPaymentIntent(ctx, pi.ID)
    if err != nil || inv == nil {
        return
    }
    if inv.Status == invoice.StatusPaid {
        return // idempotent — already handled
    }

    _ = h.invoiceStore.MarkPaid(ctx, invoice.MarkPaidParams{
        InvoiceID:       inv.ID,
        StripeChargeID:  pi.LatestCharge,
        AmountPaidCents: pi.AmountReceived,
        PaidAt:          time.Now(),
    })
}
```

### Webhook Registration

Register in Stripe Dashboard (or via Stripe CLI for local dev):
- Endpoint: `https://manifest.fireflysoftware.dev/webhooks/stripe`
- Events to listen for:
  - `payment_intent.succeeded`
  - `payment_intent.payment_failed`

---

## New Store Methods (invoice package)

```go
// SetPaymentIntent persists the Stripe PaymentIntent ID to the invoice
SetPaymentIntent(ctx context.Context, invoiceID int64, intentID string) error

// GetByPaymentIntent looks up an invoice by its Stripe PaymentIntent ID
GetByPaymentIntent(ctx context.Context, intentID string) (*Invoice, error)

// MarkPaid transitions invoice to paid and records payment details
MarkPaid(ctx context.Context, params MarkPaidParams) error

type MarkPaidParams struct {
    InvoiceID       int64
    StripeChargeID  string
    AmountPaidCents int64
    PaidAt          time.Time
}
```

---

## Routes (Phase 3 additions)

```
# Public
GET  /i/{token}/confirmed         → invoice.PaymentConfirmed   (thank-you page)

# Webhooks (no auth, verified by Stripe signature)
POST /webhooks/stripe             → payment.StripeWebhook
```

---

## Local Development

Use the Stripe CLI to forward webhooks to your local server:

```bash
stripe listen --forward-to localhost:8080/webhooks/stripe
```

The CLI outputs a webhook signing secret — use that as `STRIPE_WEBHOOK_SECRET` in your local `.env`.

For testing ACH, use Stripe's test bank account numbers:
- Routing: `110000000`
- Account: `000123456789`

---

## Phase 3 Checklist

- [ ] Run migration 00008
- [ ] Add `stripe-go/v84` dependency
- [ ] Add `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET` to `.env`
- [ ] Store Stripe publishable key in settings table
- [ ] Implement `payment` package (CreateOrGetIntent, TotalCents)
- [ ] Update public invoice handler to create PaymentIntent and pass client_secret to template
- [ ] Embed Stripe.js Payment Element in public invoice template
- [ ] Implement `/i/{token}/confirmed` thank-you page
- [ ] Implement webhook handler with signature verification
- [ ] Implement `handlePaymentSucceeded` — mark invoice paid
- [ ] Register webhook in Stripe Dashboard
- [ ] Test with Stripe CLI: card payment, ACH payment, duplicate webhook (idempotency)
- [ ] Test: paid invoice page shows confirmation instead of payment form

---

## What's Not In This Phase

- Email delivery (considered for future — send the link manually for now)
- Refund handling
- Expenses (Phase 4)
- Reporting (Phase 5)
