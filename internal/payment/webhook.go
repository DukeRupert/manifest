package payment

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"fireflysoftware.dev/manifest/internal/invoice"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

type WebhookHandler struct {
	invoiceStore  *invoice.Store
	webhookSecret string
}

func NewWebhookHandler(invoiceStore *invoice.Store, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		invoiceStore:  invoiceStore,
		webhookSecret: webhookSecret,
	}
}

func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("payment_intent.payment_failed received")
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handlePaymentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) {
	inv, err := h.invoiceStore.GetByPaymentIntent(ctx, pi.ID)
	if err != nil || inv == nil {
		log.Printf("webhook: invoice not found for payment intent %s", pi.ID)
		return
	}
	if inv.Status == invoice.StatusPaid {
		return // idempotent
	}

	chargeID := ""
	if pi.LatestCharge != nil {
		chargeID = pi.LatestCharge.ID
	}

	err = h.invoiceStore.MarkPaid(ctx, invoice.MarkPaidParams{
		InvoiceID:       inv.ID,
		StripeChargeID:  chargeID,
		AmountPaidCents: pi.AmountReceived,
		PaidAt:          time.Now(),
	})
	if err != nil {
		log.Printf("webhook: mark paid failed for invoice %d: %v", inv.ID, err)
	}
}
