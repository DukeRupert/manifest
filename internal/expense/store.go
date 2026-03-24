package expense

import (
	"context"
	"fmt"
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

// --- Categories ---

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	orgID := auth.OrgID(ctx)
	rows, err := s.pool.Query(ctx,
		`SELECT uuid, org_id, name, created_at FROM expense_categories WHERE org_id = $1 ORDER BY name ASC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *Store) GetCategory(ctx context.Context, uuid string) (*Category, error) {
	orgID := auth.OrgID(ctx)
	var c Category
	err := s.pool.QueryRow(ctx,
		`SELECT uuid, org_id, name, created_at FROM expense_categories WHERE uuid = $1 AND org_id = $2`, uuid, orgID,
	).Scan(&c.ID, &c.OrgID, &c.Name, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateCategory(ctx context.Context, name string) (*Category, error) {
	orgID := auth.OrgID(ctx)
	var c Category
	err := s.pool.QueryRow(ctx,
		`INSERT INTO expense_categories (uuid, org_id, name) VALUES (gen_random_uuid(), $1, $2)
		 RETURNING uuid, org_id, name, created_at`,
		orgID, name,
	).Scan(&c.ID, &c.OrgID, &c.Name, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpdateCategory(ctx context.Context, uuid string, name string) error {
	orgID := auth.OrgID(ctx)
	_, err := s.pool.Exec(ctx,
		`UPDATE expense_categories SET name = $2 WHERE uuid = $1 AND org_id = $3`, uuid, name, orgID)
	return err
}

func (s *Store) DeleteCategory(ctx context.Context, uuid string) error {
	orgID := auth.OrgID(ctx)
	// Guard: don't delete if expenses reference it
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM expenses e
		 JOIN expense_categories ec ON ec.id = e.category_id
		 WHERE ec.uuid = $1 AND ec.org_id = $2`, uuid, orgID,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete category: %d expenses reference it", count)
	}
	_, err = s.pool.Exec(ctx,
		`DELETE FROM expense_categories WHERE uuid = $1 AND org_id = $2`, uuid, orgID)
	return err
}

// --- Expenses ---

type ListFilter struct {
	From       *time.Time
	To         *time.Time
	CategoryID *string // UUID
}

func (s *Store) Create(ctx context.Context, e *Expense) error {
	orgID := auth.OrgID(ctx)
	e.OrgID = orgID
	// Look up category internal ID from UUID
	var catInternalID int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM expense_categories WHERE uuid = $1 AND org_id = $2`, e.CategoryID, orgID,
	).Scan(&catInternalID)
	if err != nil {
		return fmt.Errorf("category lookup: %w", err)
	}
	return s.pool.QueryRow(ctx,
		`INSERT INTO expenses (uuid, org_id, category_id, vendor, amount, notes, date)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
		 RETURNING uuid, created_at, updated_at`,
		orgID, catInternalID, e.Vendor, e.Amount, e.Notes, e.Date,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (s *Store) Get(ctx context.Context, uuid string) (*Expense, error) {
	orgID := auth.OrgID(ctx)
	var e Expense
	err := s.pool.QueryRow(ctx, `
		SELECT e.uuid, e.org_id, ec.uuid, e.vendor, e.amount, e.notes, e.date, e.created_at, e.updated_at,
		       ec.uuid, ec.name
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.uuid = $1 AND e.org_id = $2
	`, uuid, orgID).Scan(
		&e.ID, &e.OrgID, &e.CategoryID, &e.Vendor, &e.Amount, &e.Notes, &e.Date, &e.CreatedAt, &e.UpdatedAt,
		&e.Category.ID, &e.Category.Name,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) List(ctx context.Context, filter ListFilter) ([]Expense, error) {
	orgID := auth.OrgID(ctx)
	query := `
		SELECT e.uuid, e.org_id, ec.uuid, e.vendor, e.amount, e.notes, e.date, e.created_at, e.updated_at,
		       ec.uuid, ec.name
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.org_id = $1`
	args := []any{orgID}
	argN := 2

	if filter.From != nil {
		query += fmt.Sprintf(` AND e.date >= $%d`, argN)
		args = append(args, *filter.From)
		argN++
	}
	if filter.To != nil {
		query += fmt.Sprintf(` AND e.date <= $%d`, argN)
		args = append(args, *filter.To)
		argN++
	}
	if filter.CategoryID != nil {
		query += fmt.Sprintf(` AND ec.uuid = $%d`, argN)
		args = append(args, *filter.CategoryID)
		argN++
	}

	query += ` ORDER BY e.date DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.CategoryID, &e.Vendor, &e.Amount, &e.Notes, &e.Date, &e.CreatedAt, &e.UpdatedAt,
			&e.Category.ID, &e.Category.Name,
		); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

func (s *Store) Update(ctx context.Context, e *Expense) error {
	orgID := auth.OrgID(ctx)
	// Look up category internal ID from UUID
	var catInternalID int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM expense_categories WHERE uuid = $1 AND org_id = $2`, e.CategoryID, orgID,
	).Scan(&catInternalID)
	if err != nil {
		return fmt.Errorf("category lookup: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE expenses
		SET category_id = $2, vendor = $3, amount = $4, notes = $5, date = $6, updated_at = NOW()
		WHERE uuid = $1 AND org_id = $7`,
		e.ID, catInternalID, e.Vendor, e.Amount, e.Notes, e.Date, orgID)
	return err
}

func (s *Store) Delete(ctx context.Context, uuid string) error {
	orgID := auth.OrgID(ctx)
	_, err := s.pool.Exec(ctx,
		`DELETE FROM expenses WHERE uuid = $1 AND org_id = $2`, uuid, orgID)
	return err
}
