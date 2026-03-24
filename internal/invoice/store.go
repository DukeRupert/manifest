package invoice

import (
	"context"
	"fmt"
	"strings"

	"fireflysoftware.dev/manifest/internal/auth"
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
func (s *Store) NextInvoiceNumber(ctx context.Context, tx pgx.Tx, orgID string, clientInternalID int64, slug string) (string, error) {
	var next int64
	err := tx.QueryRow(ctx, `
		INSERT INTO invoice_sequences (org_id, client_id, next_val)
		VALUES ($1, $2, 2)
		ON CONFLICT (org_id, client_id) DO UPDATE
			SET next_val = invoice_sequences.next_val + 1
		RETURNING next_val - 1
	`, orgID, clientInternalID).Scan(&next)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INV-%s-%04d", strings.ToUpper(slug), next), nil
}

// Create inserts an invoice and its line items in a single transaction.
func (s *Store) Create(ctx context.Context, inv *Invoice) error {
	orgID := auth.OrgID(ctx)
	inv.OrgID = orgID

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Look up client's internal ID and slug from UUID
	var clientInternalID int64
	var slug string
	err = tx.QueryRow(ctx,
		`SELECT id, slug FROM clients WHERE uuid = $1 AND org_id = $2`,
		inv.ClientID, orgID,
	).Scan(&clientInternalID, &slug)
	if err != nil {
		return fmt.Errorf("client lookup: %w", err)
	}

	number, err := s.NextInvoiceNumber(ctx, tx, orgID, clientInternalID, slug)
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
		INSERT INTO invoices (uuid, org_id, number, client_id, status, tax_rate, notes, due_date, issued_at, view_token)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING uuid, created_at, updated_at
	`, orgID, inv.Number, clientInternalID, StatusDraft, inv.TaxRate, inv.Notes, inv.DueDate, inv.IssuedAt, inv.ViewToken,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}

	// Get the BIGSERIAL id for line item FK
	var invoiceInternalID int64
	err = tx.QueryRow(ctx, `SELECT id FROM invoices WHERE uuid = $1`, inv.ID).Scan(&invoiceInternalID)
	if err != nil {
		return fmt.Errorf("invoice internal id: %w", err)
	}

	for i := range inv.LineItems {
		li := &inv.LineItems[i]
		li.Position = i
		err = tx.QueryRow(ctx, `
			INSERT INTO invoice_line_items (uuid, invoice_id, description, quantity, unit_price, position)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
			RETURNING uuid
		`, invoiceInternalID, li.Description, li.Quantity, li.UnitPrice, li.Position,
		).Scan(&li.ID)
		if err != nil {
			return fmt.Errorf("insert line item %d: %w", i, err)
		}
		li.InvoiceID = inv.ID
	}

	return tx.Commit(ctx)
}

// Get retrieves an invoice by UUID with its client and line items.
func (s *Store) Get(ctx context.Context, uuid string) (*Invoice, error) {
	var inv Invoice
	var clientInternalID int64
	err := s.pool.QueryRow(ctx, `
		SELECT i.uuid, i.org_id, i.number, c.uuid, i.status, i.tax_rate, i.notes, i.due_date,
		       i.issued_at, i.paid_at, i.view_token,
		       i.stripe_payment_intent_id, i.stripe_charge_id, i.amount_paid_cents,
		       i.created_at, i.updated_at,
		       c.id, c.name, c.slug, c.email, c.phone, c.billing_address
		FROM invoices i
		JOIN clients c ON c.id = i.client_id
		WHERE i.uuid = $1
	`, uuid).Scan(
		&inv.ID, &inv.OrgID, &inv.Number, &inv.ClientID, &inv.Status, &inv.TaxRate, &inv.Notes, &inv.DueDate,
		&inv.IssuedAt, &inv.PaidAt, &inv.ViewToken,
		&inv.StripePaymentIntentID, &inv.StripeChargeID, &inv.AmountPaidCents,
		&inv.CreatedAt, &inv.UpdatedAt,
		&clientInternalID, &inv.Client.Name, &inv.Client.Slug, &inv.Client.Email,
		&inv.Client.Phone, &inv.Client.BillingAddress,
	)
	if err != nil {
		return nil, err
	}
	inv.Client.ID = inv.ClientID
	inv.Client.InternalID = clientInternalID

	inv.LineItems, err = s.getLineItems(ctx, uuid)
	if err != nil {
		return nil, err
	}

	return &inv, nil
}

// GetByToken retrieves an invoice by its public view token (globally unique, no org scoping).
func (s *Store) GetByToken(ctx context.Context, token string) (*Invoice, error) {
	var uuid string
	err := s.pool.QueryRow(ctx,
		`SELECT uuid FROM invoices WHERE view_token = $1`, token,
	).Scan(&uuid)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, uuid)
}

func (s *Store) getLineItems(ctx context.Context, invoiceUUID string) ([]LineItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT li.uuid, $1, li.description, li.quantity, li.unit_price, li.position
		FROM invoice_line_items li
		JOIN invoices i ON i.id = li.invoice_id
		WHERE i.uuid = $1
		ORDER BY li.position ASC
	`, invoiceUUID)
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
// If status is non-empty, filters to that status only.
func (s *Store) List(ctx context.Context, status Status) ([]InvoiceListItem, error) {
	orgID := auth.OrgID(ctx)
	query := `
		SELECT
			i.uuid,
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
		WHERE i.org_id = $1`

	args := []any{orgID}
	if status != "" {
		query += ` AND i.status = $2`
		args = append(args, status)
	}

	query += `
		GROUP BY i.id, c.name
		ORDER BY i.created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
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
	orgID := auth.OrgID(ctx)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Verify invoice is still draft and belongs to this org
	var status Status
	var invoiceInternalID int64
	err = tx.QueryRow(ctx,
		`SELECT status, id FROM invoices WHERE uuid = $1 AND org_id = $2`, inv.ID, orgID,
	).Scan(&status, &invoiceInternalID)
	if err != nil {
		return err
	}
	if status != StatusDraft {
		return fmt.Errorf("can only edit draft invoices (current status: %s)", status)
	}

	_, err = tx.Exec(ctx, `
		UPDATE invoices
		SET tax_rate = $2, notes = $3, due_date = $4, updated_at = NOW()
		WHERE uuid = $1 AND org_id = $5
	`, inv.ID, inv.TaxRate, inv.Notes, inv.DueDate, orgID)
	if err != nil {
		return err
	}

	// Replace line items
	_, err = tx.Exec(ctx, `DELETE FROM invoice_line_items WHERE invoice_id = $1`, invoiceInternalID)
	if err != nil {
		return err
	}

	for i := range inv.LineItems {
		li := &inv.LineItems[i]
		li.Position = i
		err = tx.QueryRow(ctx, `
			INSERT INTO invoice_line_items (uuid, invoice_id, description, quantity, unit_price, position)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
			RETURNING uuid
		`, invoiceInternalID, li.Description, li.Quantity, li.UnitPrice, li.Position,
		).Scan(&li.ID)
		if err != nil {
			return err
		}
		li.InvoiceID = inv.ID
	}

	return tx.Commit(ctx)
}

// Transition changes the invoice status if the transition is valid.
func (s *Store) Transition(ctx context.Context, invoiceUUID string, to Status) error {
	var current Status
	err := s.pool.QueryRow(ctx,
		`SELECT status FROM invoices WHERE uuid = $1`, invoiceUUID,
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

	return s.setStatus(ctx, invoiceUUID, to)
}

func (s *Store) setStatus(ctx context.Context, invoiceUUID string, status Status) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET status = $2, updated_at = NOW() WHERE uuid = $1`,
		invoiceUUID, status,
	)
	return err
}

// SetPaymentIntent persists the Stripe PaymentIntent ID to the invoice.
func (s *Store) SetPaymentIntent(ctx context.Context, invoiceUUID string, intentID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET stripe_payment_intent_id = $2, updated_at = NOW() WHERE uuid = $1`,
		invoiceUUID, intentID,
	)
	return err
}

// GetByPaymentIntent looks up an invoice by its Stripe PaymentIntent ID (globally unique).
func (s *Store) GetByPaymentIntent(ctx context.Context, intentID string) (*Invoice, error) {
	var uuid string
	err := s.pool.QueryRow(ctx,
		`SELECT uuid FROM invoices WHERE stripe_payment_intent_id = $1`, intentID,
	).Scan(&uuid)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, uuid)
}

// MarkPaid transitions an invoice to paid and records payment details.
func (s *Store) MarkPaid(ctx context.Context, params MarkPaidParams) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE invoices
		SET status = 'paid', stripe_charge_id = $2, amount_paid_cents = $3,
		    paid_at = $4, updated_at = NOW()
		WHERE uuid = $1
	`, params.InvoiceID, params.StripeChargeID, params.AmountPaidCents, params.PaidAt)
	return err
}
