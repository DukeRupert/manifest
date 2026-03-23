package reports

import (
	"net/http"
	"strconv"
	"time"

	"fireflysoftware.dev/manifest/templates"
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

	v := &templates.DashboardSummaryView{
		MonthRevenue: d.MonthRevenue,
		ARCount:      d.ARCount,
		ARTotal:      d.ARTotal,
		YTDRevenue:   d.YTDRevenue,
		YTDExpenses:  d.YTDExpenses,
		YTDNet:       d.YTDNet,
	}
	templates.ReportsIndex(v).Render(r.Context(), w)
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
	views := make([]templates.RevenueRowView, len(rows))
	for i, row := range rows {
		ytd += row.Revenue
		views[i] = templates.RevenueRowView{
			Month:        row.Month.Format("January 2006"),
			InvoiceCount: row.InvoiceCount,
			Revenue:      row.Revenue,
		}
	}

	data := &templates.RevenueReportView{
		Year:     year,
		Rows:     views,
		YTDTotal: ytd,
	}
	templates.ReportsRevenue(data).Render(r.Context(), w)
}

func (h *Handler) AR(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.OutstandingAR(r.Context())
	if err != nil {
		http.Error(w, "failed to load AR", http.StatusInternalServerError)
		return
	}

	var total float64
	views := make([]templates.ARRowView, len(rows))
	for i, row := range rows {
		total += row.Total
		dueDate := "—"
		if row.DueDate != nil {
			dueDate = row.DueDate.Format("Jan 2, 2006")
		}
		views[i] = templates.ARRowView{
			Number:      row.Number,
			ClientName:  row.ClientName,
			DueDate:     dueDate,
			IssuedAt:    row.IssuedAt.Format("Jan 2, 2006"),
			DaysOverdue: row.DaysOverdue,
			Total:       row.Total,
		}
	}

	data := &templates.ARReportView{
		Rows:  views,
		Total: total,
		Count: len(rows),
	}
	templates.ReportsAR(data).Render(r.Context(), w)
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
			to = parsed.AddDate(0, 0, 1) // inclusive end date
		}
	}

	rows, err := h.store.ProfitAndLoss(r.Context(), from, to)
	if err != nil {
		http.Error(w, "failed to load P&L", http.StatusInternalServerError)
		return
	}

	var totalRev, totalExp, totalNet float64
	views := make([]templates.PLRowView, len(rows))
	for i, row := range rows {
		totalRev += row.Revenue
		totalExp += row.Expenses
		totalNet += row.Net
		views[i] = templates.PLRowView{
			Month:    row.Month.Format("January 2006"),
			Revenue:  row.Revenue,
			Expenses: row.Expenses,
			Net:      row.Net,
		}
	}

	data := &templates.PLReportView{
		Rows:         views,
		TotalRevenue: totalRev,
		TotalExpense: totalExp,
		TotalNet:     totalNet,
		FilterFrom:   from.Format("2006-01-02"),
		FilterTo:     to.AddDate(0, 0, -1).Format("2006-01-02"),
	}
	templates.ReportsPL(data).Render(r.Context(), w)
}
