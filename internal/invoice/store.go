package invoice

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validTransitions = map[Status][]Status{
	StatusDraft:  {StatusSent, StatusVoid},
	StatusSent:   {StatusViewed, StatusPaid, StatusVoid},
	StatusViewed: {StatusPaid, StatusVoid},
	StatusPaid:   {},
	StatusVoid:   {},
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// NextInvoiceNumber atomically generates the next invoice number for a client.
// Must be called within the provided transaction.
func (s *Store) NextInvoiceNumber(ctx context.Context, tx pgx.Tx, clientID int64, slug string) (string, error) {
	var next int64
	err := tx.QueryRow(ctx, `
		INSERT INTO invoice_sequences (client_id, next_val)
		VALUES ($1, 2)
		ON CONFLICT (client_id) DO UPDATE
			SET next_val = invoice_sequences.next_val + 1
		RETURNING next_val - 1
	`, clientID).Scan(&next)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INV-%s-%04d", strings.ToUpper(slug), next), nil
}

// Create inserts an invoice and its line items in a single transaction.
func (s *Store) Create(ctx context.Context, inv *Invoice) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get client slug for invoice number
	var slug string
	err = tx.QueryRow(ctx, `SELECT slug FROM clients WHERE id = $1`, inv.ClientID).Scan(&slug)
	if err != nil {
		return fmt.Errorf("client lookup: %w", err)
	}

	number, err := s.NextInvoiceNumber(ctx, tx, inv.ClientID, slug)
	if err != nil {
		return fmt.Errorf("invoice number: %w", err)
	}
	inv.Number = number

	viewToken, err := GenerateViewToken()
	if err != nil {
		return fmt.Errorf("view token: %w", err)
	}
	inv.ViewToken = viewToken

	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (number, client_id, status, tax_rate, notes, due_date, issued_at, view_token)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, inv.Number, inv.ClientID, StatusDraft, inv.TaxRate, inv.Notes, inv.DueDate, inv.IssuedAt, inv.ViewToken,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}

	for i := range inv.LineItems {
		li := &inv.LineItems[i]
		li.InvoiceID = inv.ID
		li.Position = i
		err = tx.QueryRow(ctx, `
			INSERT INTO invoice_line_items (invoice_id, description, quantity, unit_price, position)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, li.InvoiceID, li.Description, li.Quantity, li.UnitPrice, li.Position,
		).Scan(&li.ID)
		if err != nil {
			return fmt.Errorf("insert line item %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

// Get retrieves an invoice by ID with its client and line items.
func (s *Store) Get(ctx context.Context, id int64) (*Invoice, error) {
	var inv Invoice
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.number, i.client_id, i.status, i.tax_rate, i.notes, i.due_date,
		       i.issued_at, i.paid_at, i.view_token, i.created_at, i.updated_at,
		       c.id, c.name, c.slug, c.email, c.phone, c.billing_address
		FROM invoices i
		JOIN clients c ON c.id = i.client_id
		WHERE i.id = $1
	`, id).Scan(
		&inv.ID, &inv.Number, &inv.ClientID, &inv.Status, &inv.TaxRate, &inv.Notes, &inv.DueDate,
		&inv.IssuedAt, &inv.PaidAt, &inv.ViewToken, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.Client.ID, &inv.Client.Name, &inv.Client.Slug, &inv.Client.Email,
		&inv.Client.Phone, &inv.Client.BillingAddress,
	)
	if err != nil {
		return nil, err
	}

	inv.LineItems, err = s.getLineItems(ctx, inv.ID)
	if err != nil {
		return nil, err
	}

	return &inv, nil
}

// GetByToken retrieves an invoice by its public view token.
func (s *Store) GetByToken(ctx context.Context, token string) (*Invoice, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM invoices WHERE view_token = $1`, token,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Store) getLineItems(ctx context.Context, invoiceID int64) ([]LineItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, invoice_id, description, quantity, unit_price, position
		FROM invoice_line_items
		WHERE invoice_id = $1
		ORDER BY position ASC
	`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []LineItem
	for rows.Next() {
		var li LineItem
		if err := rows.Scan(&li.ID, &li.InvoiceID, &li.Description, &li.Quantity, &li.UnitPrice, &li.Position); err != nil {
			return nil, err
		}
		items = append(items, li)
	}
	return items, rows.Err()
}

// List returns all invoices with computed totals for the list view.
func (s *Store) List(ctx context.Context) ([]InvoiceListItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			i.id,
			i.number,
			c.name AS client_name,
			i.status,
			i.due_date,
			i.issued_at,
			COALESCE(SUM(li.quantity * li.unit_price), 0) AS subtotal,
			COALESCE(SUM(li.quantity * li.unit_price), 0) * (i.tax_rate / 100) AS tax,
			COALESCE(SUM(li.quantity * li.unit_price), 0) * (1 + i.tax_rate / 100) AS total
		FROM invoices i
		JOIN clients c ON c.id = i.client_id
		LEFT JOIN invoice_line_items li ON li.invoice_id = i.id
		GROUP BY i.id, c.name
		ORDER BY i.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InvoiceListItem
	for rows.Next() {
		var item InvoiceListItem
		if err := rows.Scan(&item.ID, &item.Number, &item.ClientName, &item.Status,
			&item.DueDate, &item.IssuedAt, &item.Subtotal, &item.Tax, &item.Total); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Update modifies an invoice's editable fields and replaces line items.
// Only allowed for draft invoices.
func (s *Store) Update(ctx context.Context, inv *Invoice) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Verify invoice is still draft
	var status Status
	err = tx.QueryRow(ctx, `SELECT status FROM invoices WHERE id = $1`, inv.ID).Scan(&status)
	if err != nil {
		return err
	}
	if status != StatusDraft {
		return fmt.Errorf("can only edit draft invoices (current status: %s)", status)
	}

	_, err = tx.Exec(ctx, `
		UPDATE invoices
		SET tax_rate = $2, notes = $3, due_date = $4, updated_at = NOW()
		WHERE id = $1
	`, inv.ID, inv.TaxRate, inv.Notes, inv.DueDate)
	if err != nil {
		return err
	}

	// Replace line items
	_, err = tx.Exec(ctx, `DELETE FROM invoice_line_items WHERE invoice_id = $1`, inv.ID)
	if err != nil {
		return err
	}

	for i := range inv.LineItems {
		li := &inv.LineItems[i]
		li.InvoiceID = inv.ID
		li.Position = i
		err = tx.QueryRow(ctx, `
			INSERT INTO invoice_line_items (invoice_id, description, quantity, unit_price, position)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, li.InvoiceID, li.Description, li.Quantity, li.UnitPrice, li.Position,
		).Scan(&li.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Transition changes the invoice status if the transition is valid.
func (s *Store) Transition(ctx context.Context, invoiceID int64, to Status) error {
	var current Status
	err := s.pool.QueryRow(ctx,
		`SELECT status FROM invoices WHERE id = $1`, invoiceID,
	).Scan(&current)
	if err != nil {
		return err
	}

	allowed := validTransitions[current]
	valid := false
	for _, a := range allowed {
		if a == to {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid transition: %s → %s", current, to)
	}

	return s.setStatus(ctx, invoiceID, to)
}

func (s *Store) setStatus(ctx context.Context, invoiceID int64, status Status) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET status = $2, updated_at = NOW() WHERE id = $1`,
		invoiceID, status,
	)
	return err
}
