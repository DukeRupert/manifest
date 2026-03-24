package client

import "time"

type Client struct {
	ID             string // UUID
	OrgID          string // UUID
	InternalID     int64  // BIGSERIAL — used only for FK references, never exposed
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
