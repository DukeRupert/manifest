package expense

import (
	"context"
	"fmt"
	"time"

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
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, created_at FROM expense_categories ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *Store) GetCategory(ctx context.Context, id int64) (*Category, error) {
	var c Category
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, created_at FROM expense_categories WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateCategory(ctx context.Context, name string) (*Category, error) {
	var c Category
	err := s.pool.QueryRow(ctx,
		`INSERT INTO expense_categories (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpdateCategory(ctx context.Context, id int64, name string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE expense_categories SET name = $2 WHERE id = $1`, id, name)
	return err
}

func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	// Guard: don't delete if expenses reference it
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM expenses WHERE category_id = $1`, id,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete category: %d expenses reference it", count)
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1`, id)
	return err
}

// --- Expenses ---

type ListFilter struct {
	From       *time.Time
	To         *time.Time
	CategoryID *int64
}

func (s *Store) Create(ctx context.Context, e *Expense) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO expenses (category_id, vendor, amount, notes, date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		e.CategoryID, e.Vendor, e.Amount, e.Notes, e.Date,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (s *Store) Get(ctx context.Context, id int64) (*Expense, error) {
	var e Expense
	err := s.pool.QueryRow(ctx, `
		SELECT e.id, e.category_id, e.vendor, e.amount, e.notes, e.date, e.created_at, e.updated_at,
		       ec.id, ec.name
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.id = $1
	`, id).Scan(
		&e.ID, &e.CategoryID, &e.Vendor, &e.Amount, &e.Notes, &e.Date, &e.CreatedAt, &e.UpdatedAt,
		&e.Category.ID, &e.Category.Name,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) List(ctx context.Context, filter ListFilter) ([]Expense, error) {
	query := `
		SELECT e.id, e.category_id, e.vendor, e.amount, e.notes, e.date, e.created_at, e.updated_at,
		       ec.id, ec.name
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE 1=1`
	args := []any{}
	argN := 1

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
		query += fmt.Sprintf(` AND e.category_id = $%d`, argN)
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
			&e.ID, &e.CategoryID, &e.Vendor, &e.Amount, &e.Notes, &e.Date, &e.CreatedAt, &e.UpdatedAt,
			&e.Category.ID, &e.Category.Name,
		); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

func (s *Store) Update(ctx context.Context, e *Expense) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE expenses
		SET category_id = $2, vendor = $3, amount = $4, notes = $5, date = $6, updated_at = NOW()
		WHERE id = $1`,
		e.ID, e.CategoryID, e.Vendor, e.Amount, e.Notes, e.Date)
	return err
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id)
	return err
}
