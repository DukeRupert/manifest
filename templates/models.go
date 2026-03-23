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

// --- Reports view models ---

// DashboardSummaryView holds the reports index data.
type DashboardSummaryView struct {
	MonthRevenue float64
	ARCount      int
	ARTotal      float64
	YTDRevenue   float64
	YTDExpenses  float64
	YTDNet       float64
}

func (d DashboardSummaryView) FormatMonthRevenue() string { return fmt.Sprintf("$%.2f", d.MonthRevenue) }
func (d DashboardSummaryView) FormatARTotal() string      { return fmt.Sprintf("$%.2f", d.ARTotal) }
func (d DashboardSummaryView) FormatYTDRevenue() string   { return fmt.Sprintf("$%.2f", d.YTDRevenue) }
func (d DashboardSummaryView) FormatYTDExpenses() string  { return fmt.Sprintf("$%.2f", d.YTDExpenses) }
func (d DashboardSummaryView) FormatYTDNet() string       { return fmt.Sprintf("$%.2f", d.YTDNet) }

// RevenueRowView is a single month in the revenue report.
type RevenueRowView struct {
	Month        string
	InvoiceCount int
	Revenue      float64
}

func (r RevenueRowView) FormatRevenue() string { return fmt.Sprintf("$%.2f", r.Revenue) }

// RevenueReportView holds the full revenue report page data.
type RevenueReportView struct {
	Year     int
	Rows     []RevenueRowView
	YTDTotal float64
}

func (r RevenueReportView) FormatYTD() string { return fmt.Sprintf("$%.2f", r.YTDTotal) }

// ARRowView is a single outstanding invoice.
type ARRowView struct {
	Number      string
	ClientName  string
	DueDate     string
	IssuedAt    string
	DaysOverdue *int
	Total       float64
}

func (a ARRowView) FormatTotal() string { return fmt.Sprintf("$%.2f", a.Total) }
func (a ARRowView) IsOverdue() bool     { return a.DaysOverdue != nil && *a.DaysOverdue > 0 }
func (a ARRowView) FormatDaysOverdue() string {
	if a.DaysOverdue == nil {
		return "—"
	}
	return fmt.Sprintf("%d", *a.DaysOverdue)
}

// ARReportView holds the full AR report page data.
type ARReportView struct {
	Rows  []ARRowView
	Total float64
	Count int
}

func (a ARReportView) FormatTotal() string { return fmt.Sprintf("$%.2f", a.Total) }

// PLRowView is a single month in the P&L report.
type PLRowView struct {
	Month    string
	Revenue  float64
	Expenses float64
	Net      float64
}

func (p PLRowView) FormatRevenue() string  { return fmt.Sprintf("$%.2f", p.Revenue) }
func (p PLRowView) FormatExpenses() string { return fmt.Sprintf("$%.2f", p.Expenses) }
func (p PLRowView) FormatNet() string      { return fmt.Sprintf("$%.2f", p.Net) }
func (p PLRowView) IsNegative() bool       { return p.Net < 0 }

// PLReportView holds the full P&L report page data.
type PLReportView struct {
	Rows         []PLRowView
	TotalRevenue float64
	TotalExpense float64
	TotalNet     float64
	FilterFrom   string
	FilterTo     string
}

func (p PLReportView) FormatTotalRevenue() string { return fmt.Sprintf("$%.2f", p.TotalRevenue) }
func (p PLReportView) FormatTotalExpense() string { return fmt.Sprintf("$%.2f", p.TotalExpense) }
func (p PLReportView) FormatTotalNet() string     { return fmt.Sprintf("$%.2f", p.TotalNet) }
func (p PLReportView) IsNetNegative() bool        { return p.TotalNet < 0 }

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
