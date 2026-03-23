package client

import "time"

type Client struct {
	ID             int64
	Name           string
	Slug           string
	Email          string
	Phone          string
	BillingAddress string
	Notes          string
	ArchivedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
