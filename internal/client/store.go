package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"fireflysoftware.dev/manifest/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

var nonAlphaNum = regexp.MustCompile(`[^A-Z0-9]`)

// GenerateSlug derives a slug from the client name.
// Strips non-alphanumeric, uppercases, truncates to 8 chars.
// Appends a number if collision exists within the org.
func (s *Store) GenerateSlug(ctx context.Context, name string) (string, error) {
	orgID := auth.OrgID(ctx)
	base := strings.ToUpper(name)
	base = nonAlphaNum.ReplaceAllString(base, "")
	if len(base) > 8 {
		base = base[:8]
	}
	if base == "" {
		base = "CLIENT"
	}

	slug := base
	for i := 2; ; i++ {
		var exists bool
		err := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM clients WHERE slug = $1 AND org_id = $2)`, slug, orgID,
		).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s%d", base, i)
	}
}

func (s *Store) Create(ctx context.Context, c *Client) error {
	orgID := auth.OrgID(ctx)
	c.OrgID = orgID
	return s.pool.QueryRow(ctx,
		`INSERT INTO clients (uuid, org_id, name, slug, email, phone, billing_address, notes)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)
		 RETURNING uuid, id, created_at, updated_at`,
		orgID, c.Name, c.Slug, c.Email, c.Phone, c.BillingAddress, c.Notes,
	).Scan(&c.ID, &c.InternalID, &c.CreatedAt, &c.UpdatedAt)
}

func (s *Store) Get(ctx context.Context, uuid string) (*Client, error) {
	orgID := auth.OrgID(ctx)
	var c Client
	err := s.pool.QueryRow(ctx,
		`SELECT uuid, id, org_id, name, slug, email, phone, billing_address, notes,
		        archived_at, created_at, updated_at
		 FROM clients WHERE uuid = $1 AND org_id = $2`, uuid, orgID,
	).Scan(&c.ID, &c.InternalID, &c.OrgID, &c.Name, &c.Slug, &c.Email, &c.Phone,
		&c.BillingAddress, &c.Notes, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByInternalID looks up a client by its BIGSERIAL id. Used for FK joins.
func (s *Store) GetByInternalID(ctx context.Context, internalID int64) (*Client, error) {
	var c Client
	err := s.pool.QueryRow(ctx,
		`SELECT uuid, id, org_id, name, slug, email, phone, billing_address, notes,
		        archived_at, created_at, updated_at
		 FROM clients WHERE id = $1`, internalID,
	).Scan(&c.ID, &c.InternalID, &c.OrgID, &c.Name, &c.Slug, &c.Email, &c.Phone,
		&c.BillingAddress, &c.Notes, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) List(ctx context.Context) ([]Client, error) {
	orgID := auth.OrgID(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT uuid, id, org_id, name, slug, email, phone, billing_address, notes,
		        archived_at, created_at, updated_at
		 FROM clients
		 WHERE archived_at IS NULL AND org_id = $1
		 ORDER BY name ASC`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.InternalID, &c.OrgID, &c.Name, &c.Slug, &c.Email,
			&c.Phone, &c.BillingAddress, &c.Notes, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

func (s *Store) Update(ctx context.Context, c *Client) error {
	orgID := auth.OrgID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE clients SET name = $2, email = $3, phone = $4, billing_address = $5, notes = $6, updated_at = NOW()
		 WHERE uuid = $1 AND org_id = $7`,
		c.ID, c.Name, c.Email, c.Phone, c.BillingAddress, c.Notes, orgID,
	)
	return err
}

func (s *Store) Archive(ctx context.Context, uuid string) error {
	orgID := auth.OrgID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE clients SET archived_at = NOW(), updated_at = NOW() WHERE uuid = $1 AND org_id = $2`,
		uuid, orgID,
	)
	return err
}
