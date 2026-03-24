package settings

import (
	"context"

	"fireflysoftware.dev/manifest/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Settings struct {
	ID              string // UUID
	OrgID           string // UUID
	BusinessName    string
	BusinessAddress string
	BusinessEmail   string
	DefaultTaxRate  float64
	StripePK        string
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Get retrieves settings for the current org (from auth context).
func (s *Store) Get(ctx context.Context) (*Settings, error) {
	orgID := auth.OrgID(ctx)
	return s.GetByOrgID(ctx, orgID)
}

// GetByOrgID retrieves settings for a specific org. Used by public pages and webhooks
// where there is no auth context.
func (s *Store) GetByOrgID(ctx context.Context, orgID string) (*Settings, error) {
	var st Settings
	err := s.pool.QueryRow(ctx,
		`SELECT uuid, org_id, business_name, business_address, business_email, default_tax_rate, stripe_pk
		 FROM settings WHERE org_id = $1 LIMIT 1`, orgID,
	).Scan(&st.ID, &st.OrgID, &st.BusinessName, &st.BusinessAddress, &st.BusinessEmail, &st.DefaultTaxRate, &st.StripePK)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) Update(ctx context.Context, st *Settings) error {
	orgID := auth.OrgID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE settings
		 SET business_name = $1, business_address = $2, business_email = $3,
		     default_tax_rate = $4, stripe_pk = $5, updated_at = NOW()
		 WHERE uuid = $6 AND org_id = $7`,
		st.BusinessName, st.BusinessAddress, st.BusinessEmail,
		st.DefaultTaxRate, st.StripePK, st.ID, orgID,
	)
	return err
}
