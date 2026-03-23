package templates

import (
	"fmt"
	"time"
)

// ClientView is a view-layer model to avoid import cycles with internal/client.
type ClientView struct {
	ID             int64
	Name           string
	Slug           string
	Email          string
	Phone          string
	BillingAddress string
	Notes          string
	ArchivedAt     *time.Time
}

// InvoiceStatus mirrors invoice.Status for the template layer.
type InvoiceStatus string

const (
	StatusDraft  InvoiceStatus = "draft"
	StatusSent   InvoiceStatus = "sent"
	StatusViewed InvoiceStatus = "viewed"
	StatusPaid   InvoiceStatus = "paid"
	StatusVoid   InvoiceStatus = "void"
)

// StatusBadgeClass returns the badge CSS class for a given status.
func StatusBadgeClass(s InvoiceStatus) string {
	switch s {
	case StatusDraft:
		return "badge badge-cortex"
	case StatusSent:
		return "badge badge-crew"
	case StatusViewed:
		return "badge badge-gold"
	case StatusPaid:
		return "badge badge-guild"
	case StatusVoid:
		return "badge badge-wanted"
	default:
		return "badge badge-cortex"
	}
}

// InvoiceListItemView is a denormalized row for the invoice list.
type InvoiceListItemView struct {
	ID         int64
	Number     string
	ClientName string
	Status     InvoiceStatus
	DueDate    *time.Time
	IssuedAt   time.Time
	Total      float64
}

// FormatTotal returns the total as a dollar string.
func (i InvoiceListItemView) FormatTotal() string {
	return fmt.Sprintf("$%.2f", i.Total)
}

// FormatDueDate returns the due date formatted, or "—" if nil.
func (i InvoiceListItemView) FormatDueDate() string {
	if i.DueDate == nil {
		return "—"
	}
	return i.DueDate.Format("Jan 2, 2006")
}

// FormatIssuedAt returns the issued date formatted.
func (i InvoiceListItemView) FormatIssuedAt() string {
	return i.IssuedAt.Format("Jan 2, 2006")
}

// LineItemView is a view-layer line item.
type LineItemView struct {
	ID          int64
	Description string
	Quantity    float64
	UnitPrice   float64
}

// Subtotal returns quantity * unit price.
func (li LineItemView) Subtotal() float64 {
	return li.Quantity * li.UnitPrice
}

// FormatUnitPrice returns the unit price as a dollar string.
func (li LineItemView) FormatUnitPrice() string {
	return fmt.Sprintf("%.2f", li.UnitPrice)
}

// FormatQuantity returns the quantity formatted.
func (li LineItemView) FormatQuantity() string {
	return fmt.Sprintf("%.2f", li.Quantity)
}

// FormatSubtotal returns the line subtotal as a dollar string.
func (li LineItemView) FormatSubtotal() string {
	return fmt.Sprintf("$%.2f", li.Subtotal())
}

// InvoiceView is the full invoice for show/edit pages.
type InvoiceView struct {
	ID         int64
	Number     string
	ClientID   int64
	ClientName string
	ClientSlug string
	Status     InvoiceStatus
	TaxRate    float64
	Notes      string
	DueDate    *time.Time
	IssuedAt   time.Time
	PaidAt     *time.Time
	ViewToken  string
	LineItems  []LineItemView
}

// Subtotal returns sum of line item subtotals.
func (inv InvoiceView) Subtotal() float64 {
	var total float64
	for _, li := range inv.LineItems {
		total += li.Subtotal()
	}
	return total
}

// TaxAmount returns the tax portion.
func (inv InvoiceView) TaxAmount() float64 {
	return inv.Subtotal() * (inv.TaxRate / 100)
}

// Total returns subtotal + tax.
func (inv InvoiceView) Total() float64 {
	return inv.Subtotal() + inv.TaxAmount()
}

func (inv InvoiceView) FormatSubtotal() string {
	return fmt.Sprintf("$%.2f", inv.Subtotal())
}

func (inv InvoiceView) FormatTax() string {
	return fmt.Sprintf("$%.2f", inv.TaxAmount())
}

func (inv InvoiceView) FormatTotal() string {
	return fmt.Sprintf("$%.2f", inv.Total())
}

func (inv InvoiceView) FormatTaxRate() string {
	return fmt.Sprintf("%.2f", inv.TaxRate)
}

func (inv InvoiceView) FormatDueDate() string {
	if inv.DueDate == nil {
		return ""
	}
	return inv.DueDate.Format("2006-01-02")
}

func (inv InvoiceView) FormatDueDateDisplay() string {
	if inv.DueDate == nil {
		return "—"
	}
	return inv.DueDate.Format("Jan 2, 2006")
}

func (inv InvoiceView) FormatIssuedAt() string {
	return inv.IssuedAt.Format("Jan 2, 2006")
}

func (inv InvoiceView) FormatPaidAt() string {
	if inv.PaidAt == nil {
		return ""
	}
	return inv.PaidAt.Format("Jan 2, 2006")
}

// PublicInvoiceView holds everything needed for the public invoice page.
type PublicInvoiceView struct {
	Invoice         InvoiceView
	BusinessName    string
	BusinessAddress string
	BusinessEmail   string
	ClientAddress   string
	StripePK        string
	ClientSecret    string
}

// SettingsView is for the settings form.
type SettingsView struct {
	BusinessName    string
	BusinessAddress string
	BusinessEmail   string
	DefaultTaxRate  float64
	StripePK        string
}

// AllStatuses returns all invoice statuses for filter UI.
func AllStatuses() []InvoiceStatus {
	return []InvoiceStatus{StatusDraft, StatusSent, StatusViewed, StatusPaid, StatusVoid}
}

// CategoryView is a view-layer expense category.
type CategoryView struct {
	ID   int64
	Name string
}

// ExpenseView is a view-layer expense for list and detail pages.
type ExpenseView struct {
	ID           int64
	CategoryID   int64
	CategoryName string
	Vendor       string
	Amount       float64
	Notes        string
	Date         time.Time
}

func (e ExpenseView) FormatAmount() string {
	return fmt.Sprintf("$%.2f", e.Amount)
}

func (e ExpenseView) FormatDate() string {
	return e.Date.Format("Jan 2, 2006")
}

func (e ExpenseView) FormatDateInput() string {
	return e.Date.Format("2006-01-02")
}

func (e ExpenseView) FormatAmountInput() string {
	return fmt.Sprintf("%.2f", e.Amount)
}

// ExpenseListData holds everything the expense list page needs.
type ExpenseListData struct {
	Expenses   []ExpenseView
	Categories []CategoryView
	FilterFrom string
	FilterTo   string
	FilterCat  int64
	Total      float64
}

func (d ExpenseListData) FormatTotal() string {
	return fmt.Sprintf("$%.2f", d.Total)
}
