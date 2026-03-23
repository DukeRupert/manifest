package payment

import (
	"context"
	"fmt"
	"math"

	"fireflysoftware.dev/manifest/internal/invoice"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

type CreateIntentParams struct {
	InvoiceID     int64
	InvoiceNumber string
	AmountCents   int64
	ClientEmail   string
}

// TotalCents converts an invoice total from dollars to cents.
func TotalCents(inv *invoice.Invoice) int64 {
	total := inv.Total()
	return int64(math.Round(total * 100))
}

// CreateOrGetIntent creates a new Stripe PaymentIntent or retrieves an existing one.
func CreateOrGetIntent(ctx context.Context, store *invoice.Store, params CreateIntentParams) (string, error) {
	inv, err := store.Get(ctx, params.InvoiceID)
	if err != nil {
		return "", err
	}

	// If invoice already has a payment intent, retrieve its client secret
	if inv.StripePaymentIntentID != nil && *inv.StripePaymentIntentID != "" {
		pi, err := paymentintent.Get(*inv.StripePaymentIntentID, nil)
		if err != nil {
			return "", err
		}
		return pi.ClientSecret, nil
	}

	// Create new PaymentIntent
	intentParams := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(params.AmountCents),
		Currency: stripe.String("usd"),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
			"us_bank_account",
		}),
	}
	if params.ClientEmail != "" {
		intentParams.ReceiptEmail = stripe.String(params.ClientEmail)
	}
	intentParams.AddMetadata("invoice_id", fmt.Sprintf("%d", params.InvoiceID))
	intentParams.AddMetadata("invoice_number", params.InvoiceNumber)

	pi, err := paymentintent.New(intentParams)
	if err != nil {
		return "", fmt.Errorf("stripe create intent: %w", err)
	}

	// Persist intent ID to invoice
	if err := store.SetPaymentIntent(ctx, params.InvoiceID, pi.ID); err != nil {
		return "", fmt.Errorf("persist intent: %w", err)
	}

	return pi.ClientSecret, nil
}
