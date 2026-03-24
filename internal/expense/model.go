package expense

import "time"

type Category struct {
	ID        string // UUID
	OrgID     string // UUID
	Name      string
	CreatedAt time.Time
}

type Expense struct {
	ID         string // UUID
	OrgID      string // UUID
	CategoryID string // UUID
	Category   Category
	Vendor     string
	Amount     float64
	Notes      string
	Date       time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
