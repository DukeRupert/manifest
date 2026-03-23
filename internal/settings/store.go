package settings

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Settings struct {
	ID              int64
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

func (s *Store) Get(ctx context.Context) (*Settings, error) {
	var st Settings
	err := s.pool.QueryRow(ctx,
		`SELECT id, business_name, business_address, business_email, default_tax_rate, stripe_pk
		 FROM settings LIMIT 1`,
	).Scan(&st.ID, &st.BusinessName, &st.BusinessAddress, &st.BusinessEmail, &st.DefaultTaxRate, &st.StripePK)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) Update(ctx context.Context, st *Settings) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE settings
		 SET business_name = $1, business_address = $2, business_email = $3,
		     default_tax_rate = $4, stripe_pk = $5, updated_at = NOW()
		 WHERE id = $6`,
		st.BusinessName, st.BusinessAddress, st.BusinessEmail,
		st.DefaultTaxRate, st.StripePK, st.ID,
	)
	return err
}
