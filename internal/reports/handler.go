package reports

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.GetDashboardSummary(r.Context())
	if err != nil {
		http.Error(w, "failed to load dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Reports Dashboard</h1>
<div style="display:flex;gap:24px;flex-wrap:wrap">
  <div style="border:1px solid #ccc;padding:16px;min-width:200px">
    <h3>This Month Revenue</h3>
    <p style="font-size:24px"><strong>$%.2f</strong></p>
  </div>
  <div style="border:1px solid #ccc;padding:16px;min-width:200px">
    <h3>Outstanding AR</h3>
    <p style="font-size:24px"><strong>%d invoices / $%.2f</strong></p>
  </div>
  <div style="border:1px solid #ccc;padding:16px;min-width:200px">
    <h3>YTD Revenue</h3>
    <p style="font-size:24px"><strong>$%.2f</strong></p>
  </div>
  <div style="border:1px solid #ccc;padding:16px;min-width:200px">
    <h3>YTD Expenses</h3>
    <p style="font-size:24px"><strong>$%.2f</strong></p>
  </div>
  <div style="border:1px solid #ccc;padding:16px;min-width:200px">
    <h3>YTD Net</h3>
    <p style="font-size:24px"><strong>$%.2f</strong></p>
  </div>
</div>
<nav style="margin-top:24px">
  <a href="/reports/revenue">Revenue Report</a> |
  <a href="/reports/ar">Outstanding AR</a> |
  <a href="/reports/pl">P&amp;L Report</a>
</nav>`, d.MonthRevenue, d.ARCount, d.ARTotal, d.YTDRevenue, d.YTDExpenses, d.YTDNet)
}

func (h *Handler) Revenue(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	rows, err := h.store.RevenueByMonth(r.Context(), year)
	if err != nil {
		http.Error(w, "failed to load revenue", http.StatusInternalServerError)
		return
	}

	var ytd float64
	for _, r := range rows {
		ytd += r.Revenue
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Revenue Report — %d</h1>`, year)
	fmt.Fprintf(w, `<p><strong>YTD Total: $%.2f</strong></p>`, ytd)

	// Year picker
	fmt.Fprintf(w, `<form method="GET"><label>Year <input type="number" name="year" value="%d"></label> <button type="submit">Go</button></form>`, year)

	fmt.Fprintf(w, `<table><tr><th>Month</th><th>Invoices</th><th>Revenue</th></tr>`)
	for _, row := range rows {
		fmt.Fprintf(w, `<tr><td>%s</td><td>%d</td><td>$%.2f</td></tr>`,
			row.Month.Format("January 2006"), row.InvoiceCount, row.Revenue)
	}
	fmt.Fprintf(w, `</table>`)
	fmt.Fprintf(w, `<p><a href="/reports">← Back to Dashboard</a></p>`)
}

func (h *Handler) AR(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.OutstandingAR(r.Context())
	if err != nil {
		http.Error(w, "failed to load AR", http.StatusInternalServerError)
		return
	}

	var total float64
	for _, r := range rows {
		total += r.Total
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Outstanding Accounts Receivable</h1>`)
	fmt.Fprintf(w, `<p><strong>Total Outstanding: $%.2f (%d invoices)</strong></p>`, total, len(rows))

	fmt.Fprintf(w, `<table><tr><th>Invoice</th><th>Client</th><th>Issued</th><th>Due</th><th>Days Overdue</th><th>Total</th></tr>`)
	for _, row := range rows {
		due := "—"
		if row.DueDate != nil {
			due = row.DueDate.Format("2006-01-02")
		}
		overdue := "—"
		style := ""
		if row.DaysOverdue != nil {
			overdue = strconv.Itoa(*row.DaysOverdue)
			if *row.DaysOverdue > 0 {
				style = ` style="color:red;font-weight:bold"`
			}
		}
		fmt.Fprintf(w, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td%s>%s</td><td>$%.2f</td></tr>`,
			row.Number, row.ClientName, row.IssuedAt.Format("2006-01-02"), due, style, overdue, row.Total)
	}
	fmt.Fprintf(w, `</table>`)
	fmt.Fprintf(w, `<p><a href="/reports">← Back to Dashboard</a></p>`)
}

func (h *Handler) PL(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)

	if f := r.URL.Query().Get("from"); f != "" {
		if t, err := time.Parse("2006-01-02", f); err == nil {
			from = t
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			to = parsed
		}
	}

	rows, err := h.store.ProfitAndLoss(r.Context(), from, to)
	if err != nil {
		http.Error(w, "failed to load P&L", http.StatusInternalServerError)
		return
	}

	var totalRev, totalExp, totalNet float64
	for _, r := range rows {
		totalRev += r.Revenue
		totalExp += r.Expenses
		totalNet += r.Net
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Profit &amp; Loss</h1>`)

	// Date range picker
	fmt.Fprintf(w, `<form method="GET">
  <label>From <input type="date" name="from" value="%s"></label>
  <label>To <input type="date" name="to" value="%s"></label>
  <button type="submit">Filter</button>
</form>`, from.Format("2006-01-02"), to.AddDate(0, 0, -1).Format("2006-01-02"))

	fmt.Fprintf(w, `<table><tr><th>Month</th><th>Revenue</th><th>Expenses</th><th>Net</th></tr>`)
	for _, row := range rows {
		netStyle := ""
		if row.Net < 0 {
			netStyle = ` style="color:red"`
		}
		fmt.Fprintf(w, `<tr><td>%s</td><td>$%.2f</td><td>$%.2f</td><td%s>$%.2f</td></tr>`,
			row.Month.Format("January 2006"), row.Revenue, row.Expenses, netStyle, row.Net)
	}
	// YTD summary row
	netStyle := ""
	if totalNet < 0 {
		netStyle = ` style="color:red"`
	}
	fmt.Fprintf(w, `<tr style="font-weight:bold;border-top:2px solid #333"><td>Total</td><td>$%.2f</td><td>$%.2f</td><td%s>$%.2f</td></tr>`,
		totalRev, totalExp, netStyle, totalNet)
	fmt.Fprintf(w, `</table>`)
	fmt.Fprintf(w, `<p><a href="/reports">← Back to Dashboard</a></p>`)
}
