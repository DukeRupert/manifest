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
	ID          int64
	InvoiceID   int64
	Description string
	Quantity    float64
	UnitPrice   float64
	Position    int
}

func (li LineItem) Subtotal() float64 {
	return li.Quantity * li.UnitPrice
}

type Invoice struct {
	ID        int64
	Number    string
	ClientID  int64
	Client    client.Client
	Status    Status
	TaxRate   float64
	Notes     string
	DueDate   *time.Time
	IssuedAt  time.Time
	PaidAt    *time.Time
	ViewToken string
	LineItems []LineItem
	CreatedAt time.Time
	UpdatedAt time.Time
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
	ID         int64
	Number     string
	ClientName string
	Status     Status
	DueDate    *time.Time
	IssuedAt   time.Time
	Subtotal   float64
	Tax        float64
	Total      float64
}
