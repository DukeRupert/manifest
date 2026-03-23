package expense

import "time"

type Category struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type Expense struct {
	ID         int64
	CategoryID int64
	Category   Category
	Vendor     string
	Amount     float64
	Notes      string
	Date       time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
