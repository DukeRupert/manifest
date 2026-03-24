package reports

import (
	"context"
	"time"

	"fireflysoftware.dev/manifest/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// --- Dashboard Summary ---

type DashboardSummary struct {
	MonthRevenue float64
	ARCount      int
	ARTotal      float64
	YTDRevenue   float64
	YTDExpenses  float64
	YTDNet       float64
}

func (s *Store) GetDashboardSummary(ctx context.Context) (*DashboardSummary, error) {
	orgID := auth.OrgID(ctx)
	now := time.Now()
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var d DashboardSummary

	// This month revenue
	s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_paid_cents), 0) / 100.0
		FROM invoices WHERE status = 'paid' AND paid_at >= $1 AND org_id = $2
	`, monthStart, orgID).Scan(&d.MonthRevenue)

	// Outstanding AR
	s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(sub.total), 0)
		FROM (
			SELECT COALESCE(SUM(li.quantity * li.unit_price), 0) * (1 + i.tax_rate / 100) AS total
			FROM invoices i
			LEFT JOIN invoice_line_items li ON li.invoice_id = i.id
			WHERE i.status IN ('sent', 'viewed') AND i.org_id = $1
			GROUP BY i.id
		) sub
	`, orgID).Scan(&d.ARCount, &d.ARTotal)

	// YTD revenue
	s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_paid_cents), 0) / 100.0
		FROM invoices WHERE status = 'paid' AND paid_at >= $1 AND org_id = $2
	`, yearStart, orgID).Scan(&d.YTDRevenue)

	// YTD expenses
	s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE date >= $1 AND org_id = $2
	`, yearStart, orgID).Scan(&d.YTDExpenses)

	d.YTDNet = d.YTDRevenue - d.YTDExpenses

	return &d, nil
}

// --- Revenue Report ---

type RevenueRow struct {
	Month        time.Time
	InvoiceCount int
	Revenue      float64
}

func (s *Store) RevenueByMonth(ctx context.Context, year int) ([]RevenueRow, error) {
	orgID := auth.OrgID(ctx)
	rows, err := s.pool.Query(ctx, `
		SELECT DATE_TRUNC('month', paid_at) AS month,
		       COUNT(*) AS invoice_count,
		       SUM(amount_paid_cents) / 100.0 AS revenue
		FROM invoices
		WHERE status = 'paid' AND EXTRACT(YEAR FROM paid_at) = $1 AND org_id = $2
		GROUP BY month
		ORDER BY month
	`, year, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RevenueRow
	for rows.Next() {
		var r RevenueRow
		if err := rows.Scan(&r.Month, &r.InvoiceCount, &r.Revenue); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// --- Outstanding AR ---

type ARRow struct {
	Number      string
	ClientName  string
	DueDate     *time.Time
	IssuedAt    time.Time
	DaysOverdue *int
	Total       float64
}

func (s *Store) OutstandingAR(ctx context.Context) ([]ARRow, error) {
	orgID := auth.OrgID(ctx)
	rows, err := s.pool.Query(ctx, `
		SELECT
			i.number,
			c.name AS client,
			i.due_date,
			i.issued_at,
			CASE WHEN i.due_date IS NOT NULL THEN CURRENT_DATE - i.due_date END AS days_overdue,
			COALESCE(SUM(li.quantity * li.unit_price), 0) * (1 + i.tax_rate / 100) AS total
		FROM invoices i
		JOIN clients c ON c.id = i.client_id
		LEFT JOIN invoice_line_items li ON li.invoice_id = i.id
		WHERE i.status IN ('sent', 'viewed') AND i.org_id = $1
		GROUP BY i.id, c.name
		ORDER BY i.due_date ASC NULLS LAST
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ARRow
	for rows.Next() {
		var r ARRow
		if err := rows.Scan(&r.Number, &r.ClientName, &r.DueDate, &r.IssuedAt, &r.DaysOverdue, &r.Total); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// --- P&L ---

type MonthlyAmount struct {
	Month  time.Time
	Amount float64
}

type PLRow struct {
	Month    time.Time
	Revenue  float64
	Expenses float64
	Net      float64
}

func (s *Store) ProfitAndLoss(ctx context.Context, from, to time.Time) ([]PLRow, error) {
	orgID := auth.OrgID(ctx)

	// Revenue by month
	revenueMap := map[string]float64{}
	rows, err := s.pool.Query(ctx, `
		SELECT DATE_TRUNC('month', paid_at) AS month,
		       SUM(amount_paid_cents) / 100.0 AS revenue
		FROM invoices
		WHERE status = 'paid' AND paid_at >= $1 AND paid_at < $2 AND org_id = $3
		GROUP BY month
	`, from, to, orgID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m time.Time
		var amt float64
		if err := rows.Scan(&m, &amt); err != nil {
			rows.Close()
			return nil, err
		}
		revenueMap[m.Format("2006-01")] = amt
	}
	rows.Close()

	// Expenses by month
	expenseMap := map[string]float64{}
	rows, err = s.pool.Query(ctx, `
		SELECT DATE_TRUNC('month', date) AS month,
		       SUM(amount) AS expenses
		FROM expenses
		WHERE date >= $1 AND date < $2 AND org_id = $3
		GROUP BY month
	`, from, to, orgID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m time.Time
		var amt float64
		if err := rows.Scan(&m, &amt); err != nil {
			rows.Close()
			return nil, err
		}
		expenseMap[m.Format("2006-01")] = amt
	}
	rows.Close()

	// Merge into per-month P&L
	var result []PLRow
	cursor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(end) {
		key := cursor.Format("2006-01")
		rev := revenueMap[key]
		exp := expenseMap[key]
		if rev != 0 || exp != 0 {
			result = append(result, PLRow{
				Month:    cursor,
				Revenue:  rev,
				Expenses: exp,
				Net:      rev - exp,
			})
		}
		cursor = cursor.AddDate(0, 1, 0)
	}

	return result, nil
}
