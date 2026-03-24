package invoice

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"fireflysoftware.dev/manifest/internal/client"
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
	ID          string // UUID
	InvoiceID   string // UUID
	Description string
	Quantity    float64
	UnitPrice   float64
	Position    int
}

func (li LineItem) Subtotal() float64 {
	return li.Quantity * li.UnitPrice
}

type Invoice struct {
	ID        string // UUID
	OrgID     string // UUID
	Number    string
	ClientID  string // UUID (application-facing)
	Client    client.Client
	Status    Status
	TaxRate   float64
	Notes     string
	DueDate   *time.Time
	IssuedAt  time.Time
	PaidAt    *time.Time
	ViewToken string
	StripePaymentIntentID *string
	StripeChargeID        *string
	AmountPaidCents       *int64
	LineItems             []LineItem
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MarkPaidParams struct {
	InvoiceID       string // UUID
	StripeChargeID  string
	AmountPaidCents int64
	PaidAt          time.Time
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

func GenerateViewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// InvoiceListItem is a denormalized row for the invoice list view.
type InvoiceListItem struct {
	ID         string // UUID
	Number     string
	ClientName string
	Status     Status
	DueDate    *time.Time
	IssuedAt   time.Time
	Subtotal   float64
	Tax        float64
	Total      float64
}
