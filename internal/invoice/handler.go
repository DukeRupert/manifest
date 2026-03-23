package invoice

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	clientpkg "fireflysoftware.dev/manifest/internal/client"
	"fireflysoftware.dev/manifest/internal/settings"
)

// PaymentCreator creates or retrieves a Stripe PaymentIntent client secret.
// Returns empty string if Stripe is not configured.
type PaymentCreator func(ctx context.Context, inv *Invoice) (clientSecret string, err error)

type Handler struct {
	store          *Store
	clientStore    *clientpkg.Store
	settingsStore  *settings.Store
	createPayment  PaymentCreator
}

func NewHandler(store *Store, clientStore *clientpkg.Store, settingsStore *settings.Store) *Handler {
	return &Handler{
		store:         store,
		clientStore:   clientStore,
		settingsStore: settingsStore,
	}
}

func (h *Handler) SetPaymentCreator(pc PaymentCreator) {
	h.createPayment = pc
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list invoices", http.StatusInternalServerError)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Invoices (%d)</h1>`, len(items))
	fmt.Fprintf(w, `<a href="/invoices/new">New Invoice</a>`)
	fmt.Fprintf(w, `<table><tr><th>Number</th><th>Client</th><th>Status</th><th>Total</th><th>Due</th></tr>`)
	for _, item := range items {
		due := ""
		if item.DueDate != nil {
			due = item.DueDate.Format("2006-01-02")
		}
		fmt.Fprintf(w, `<tr><td><a href="/invoices/%d">%s</a></td><td>%s</td><td>%s</td><td>$%.2f</td><td>%s</td></tr>`,
			item.ID, item.Number, item.ClientName, item.Status, item.Total, due)
	}
	fmt.Fprintf(w, `</table>`)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientStore.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list clients", http.StatusInternalServerError)
		return
	}

	st, err := h.settingsStore.Get(r.Context())
	if err != nil {
		http.Error(w, "failed to load settings", http.StatusInternalServerError)
		return
	}

	// TODO: render templ template with Alpine.js for dynamic line items
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>New Invoice</h1>
<form method="POST" action="/invoices">
  <label>Client<br><select name="client_id" required>
    <option value="">Select client...</option>`)
	for _, c := range clients {
		fmt.Fprintf(w, `<option value="%d">%s</option>`, c.ID, c.Name)
	}
	fmt.Fprintf(w, `</select></label><br>
  <label>Tax Rate (%%)<br><input type="number" step="0.01" name="tax_rate" value="%.2f"></label><br>
  <label>Due Date<br><input type="date" name="due_date"></label><br>
  <label>Notes<br><textarea name="notes"></textarea></label><br>
  <h3>Line Items</h3>
  <div id="line-items">
    <div>
      <input type="text" name="li_desc[]" placeholder="Description" required>
      <input type="number" step="0.01" name="li_qty[]" value="1" min="0.01">
      <input type="number" step="0.01" name="li_price[]" placeholder="Unit Price" required>
    </div>
  </div>
  <button type="button" onclick="addLineItem()">+ Add Line Item</button><br><br>
  <button type="submit">Create Invoice</button>
</form>
<script>
function addLineItem() {
  const div = document.createElement('div');
  div.innerHTML = '<input type="text" name="li_desc[]" placeholder="Description" required> ' +
    '<input type="number" step="0.01" name="li_qty[]" value="1" min="0.01"> ' +
    '<input type="number" step="0.01" name="li_price[]" placeholder="Unit Price" required>';
  document.getElementById('line-items').appendChild(div);
}
</script>`, st.DefaultTaxRate)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid client", http.StatusBadRequest)
		return
	}

	taxRate, _ := strconv.ParseFloat(r.FormValue("tax_rate"), 64)

	inv := &Invoice{
		ClientID: clientID,
		TaxRate:  taxRate,
		Notes:    r.FormValue("notes"),
		IssuedAt: time.Now(),
	}

	if due := r.FormValue("due_date"); due != "" {
		if t, err := time.Parse("2006-01-02", due); err == nil {
			inv.DueDate = &t
		}
	}

	// Parse line items from form arrays
	descs := r.Form["li_desc[]"]
	qtys := r.Form["li_qty[]"]
	prices := r.Form["li_price[]"]
	for i := range descs {
		desc := strings.TrimSpace(descs[i])
		if desc == "" {
			continue
		}
		qty := 1.0
		if i < len(qtys) {
			if q, err := strconv.ParseFloat(qtys[i], 64); err == nil {
				qty = q
			}
		}
		price := 0.0
		if i < len(prices) {
			if p, err := strconv.ParseFloat(prices[i], 64); err == nil {
				price = p
			}
		}
		inv.LineItems = append(inv.LineItems, LineItem{
			Description: desc,
			Quantity:    qty,
			UnitPrice:   price,
		})
	}

	if err := h.store.Create(r.Context(), inv); err != nil {
		http.Error(w, fmt.Sprintf("failed to create invoice: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/invoices/%d", inv.ID), http.StatusSeeOther)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	inv, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Invoice %s</h1>`, inv.Number)
	fmt.Fprintf(w, `<p>Client: %s</p>`, inv.Client.Name)
	fmt.Fprintf(w, `<p>Status: <strong>%s</strong></p>`, inv.Status)
	fmt.Fprintf(w, `<p>Issued: %s</p>`, inv.IssuedAt.Format("2006-01-02"))
	if inv.DueDate != nil {
		fmt.Fprintf(w, `<p>Due: %s</p>`, inv.DueDate.Format("2006-01-02"))
	}

	fmt.Fprintf(w, `<table><tr><th>Description</th><th>Qty</th><th>Price</th><th>Subtotal</th></tr>`)
	for _, li := range inv.LineItems {
		fmt.Fprintf(w, `<tr><td>%s</td><td>%.2f</td><td>$%.2f</td><td>$%.2f</td></tr>`,
			li.Description, li.Quantity, li.UnitPrice, li.Subtotal())
	}
	fmt.Fprintf(w, `</table>`)

	fmt.Fprintf(w, `<p>Subtotal: $%.2f</p>`, inv.Subtotal())
	fmt.Fprintf(w, `<p>Tax (%.2f%%): $%.2f</p>`, inv.TaxRate, inv.TaxAmount())
	fmt.Fprintf(w, `<p><strong>Total: $%.2f</strong></p>`, inv.Total())

	if inv.Notes != "" {
		fmt.Fprintf(w, `<p>Notes: %s</p>`, inv.Notes)
	}

	fmt.Fprintf(w, `<p>Public URL: <a href="/i/%s">/i/%s</a></p>`, inv.ViewToken, inv.ViewToken)

	// Action buttons based on status
	if inv.Status == StatusDraft {
		fmt.Fprintf(w, `<a href="/invoices/%d/edit">Edit</a> `, inv.ID)
		fmt.Fprintf(w, `<form method="POST" action="/invoices/%d/send" style="display:inline"><button type="submit">Send</button></form> `, inv.ID)
		fmt.Fprintf(w, `<form method="POST" action="/invoices/%d/void" style="display:inline"><button type="submit">Void</button></form>`, inv.ID)
	} else if inv.Status == StatusSent || inv.Status == StatusViewed {
		fmt.Fprintf(w, `<form method="POST" action="/invoices/%d/void" style="display:inline"><button type="submit">Void</button></form>`, inv.ID)
	}
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	inv, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if inv.Status != StatusDraft {
		http.Error(w, "can only edit draft invoices", http.StatusBadRequest)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Edit %s</h1>
<form method="POST" action="/invoices/%d">
  <p>Client: %s (cannot change)</p>
  <label>Tax Rate (%%)<br><input type="number" step="0.01" name="tax_rate" value="%.2f"></label><br>`, inv.Number, inv.ID, inv.Client.Name, inv.TaxRate)

	due := ""
	if inv.DueDate != nil {
		due = inv.DueDate.Format("2006-01-02")
	}
	fmt.Fprintf(w, `<label>Due Date<br><input type="date" name="due_date" value="%s"></label><br>
  <label>Notes<br><textarea name="notes">%s</textarea></label><br>
  <h3>Line Items</h3>
  <div id="line-items">`, due, inv.Notes)

	for _, li := range inv.LineItems {
		fmt.Fprintf(w, `<div>
      <input type="text" name="li_desc[]" value="%s" required>
      <input type="number" step="0.01" name="li_qty[]" value="%.2f" min="0.01">
      <input type="number" step="0.01" name="li_price[]" value="%.2f" required>
    </div>`, li.Description, li.Quantity, li.UnitPrice)
	}

	fmt.Fprintf(w, `</div>
  <button type="button" onclick="addLineItem()">+ Add Line Item</button><br><br>
  <button type="submit">Save</button>
</form>
<script>
function addLineItem() {
  const div = document.createElement('div');
  div.innerHTML = '<input type="text" name="li_desc[]" placeholder="Description" required> ' +
    '<input type="number" step="0.01" name="li_qty[]" value="1" min="0.01"> ' +
    '<input type="number" step="0.01" name="li_price[]" placeholder="Unit Price" required>';
  document.getElementById('line-items').appendChild(div);
}
</script>`)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	taxRate, _ := strconv.ParseFloat(r.FormValue("tax_rate"), 64)

	inv := &Invoice{
		ID:      id,
		TaxRate: taxRate,
		Notes:   r.FormValue("notes"),
	}

	if due := r.FormValue("due_date"); due != "" {
		if t, err := time.Parse("2006-01-02", due); err == nil {
			inv.DueDate = &t
		}
	}

	descs := r.Form["li_desc[]"]
	qtys := r.Form["li_qty[]"]
	prices := r.Form["li_price[]"]
	for i := range descs {
		desc := strings.TrimSpace(descs[i])
		if desc == "" {
			continue
		}
		qty := 1.0
		if i < len(qtys) {
			if q, err := strconv.ParseFloat(qtys[i], 64); err == nil {
				qty = q
			}
		}
		price := 0.0
		if i < len(prices) {
			if p, err := strconv.ParseFloat(prices[i], 64); err == nil {
				price = p
			}
		}
		inv.LineItems = append(inv.LineItems, LineItem{
			Description: desc,
			Quantity:    qty,
			UnitPrice:   price,
		})
	}

	if err := h.store.Update(r.Context(), inv); err != nil {
		http.Error(w, fmt.Sprintf("failed to update invoice: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/invoices/%d", id), http.StatusSeeOther)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.store.Transition(r.Context(), id, StatusSent); err != nil {
		http.Error(w, fmt.Sprintf("failed to send: %v", err), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/invoices/%d", id), http.StatusSeeOther)
}

func (h *Handler) Void(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.store.Transition(r.Context(), id, StatusVoid); err != nil {
		http.Error(w, fmt.Sprintf("failed to void: %v", err), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/invoices/%d", id), http.StatusSeeOther)
}

// PublicView renders the public invoice page (no auth required).
func (h *Handler) PublicView(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	inv, err := h.store.GetByToken(r.Context(), token)
	if err != nil || inv == nil || inv.Status == StatusVoid {
		http.NotFound(w, r)
		return
	}

	// Auto-transition sent → viewed
	if inv.Status == StatusSent {
		_ = h.store.Transition(r.Context(), inv.ID, StatusViewed)
		inv.Status = StatusViewed
	}

	// If already paid, render paid confirmation
	if inv.Status == StatusPaid {
		h.renderPaidPage(w, inv)
		return
	}

	st, _ := h.settingsStore.Get(r.Context())

	// Create or retrieve PaymentIntent if Stripe is configured
	var clientSecret string
	stripePK := ""
	if st != nil {
		stripePK = st.StripePK
	}
	if h.createPayment != nil && stripePK != "" {
		clientSecret, err = h.createPayment(r.Context(), inv)
		if err != nil {
			// Log but don't block the page — show invoice without payment
			fmt.Printf("payment setup error: %v\n", err)
		}
	}

	// TODO: render templ template (public invoice page)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Invoice %s</title></head><body>`, inv.Number)

	if st != nil && st.BusinessName != "" {
		fmt.Fprintf(w, `<h2>%s</h2>`, st.BusinessName)
		fmt.Fprintf(w, `<p>%s</p>`, st.BusinessAddress)
	}

	fmt.Fprintf(w, `<h1>Invoice %s</h1>`, inv.Number)
	fmt.Fprintf(w, `<p>Bill to: <strong>%s</strong></p>`, inv.Client.Name)
	if inv.Client.BillingAddress != "" {
		fmt.Fprintf(w, `<p>%s</p>`, inv.Client.BillingAddress)
	}
	fmt.Fprintf(w, `<p>Issued: %s</p>`, inv.IssuedAt.Format("January 2, 2006"))
	if inv.DueDate != nil {
		fmt.Fprintf(w, `<p>Due: %s</p>`, inv.DueDate.Format("January 2, 2006"))
	}

	fmt.Fprintf(w, `<table><tr><th>Description</th><th>Qty</th><th>Price</th><th>Amount</th></tr>`)
	for _, li := range inv.LineItems {
		fmt.Fprintf(w, `<tr><td>%s</td><td>%.2f</td><td>$%.2f</td><td>$%.2f</td></tr>`,
			li.Description, li.Quantity, li.UnitPrice, li.Subtotal())
	}
	fmt.Fprintf(w, `</table>`)

	fmt.Fprintf(w, `<p>Subtotal: $%.2f</p>`, inv.Subtotal())
	fmt.Fprintf(w, `<p>Tax (%.2f%%): $%.2f</p>`, inv.TaxRate, inv.TaxAmount())
	fmt.Fprintf(w, `<p><strong>Total: $%.2f</strong></p>`, inv.Total())

	// Stripe Payment Element
	if clientSecret != "" && stripePK != "" {
		fmt.Fprintf(w, `
<hr>
<h3>Pay Now</h3>
<div id="payment-element"></div>
<button id="pay-button" style="margin-top:16px">Pay $%.2f</button>
<div id="error-message" style="color:red;margin-top:8px"></div>
<script src="https://js.stripe.com/v3/"></script>
<script>
  const stripe = Stripe('%s');
  const elements = stripe.elements({ clientSecret: '%s' });
  const paymentElement = elements.create('payment');
  paymentElement.mount('#payment-element');
  document.getElementById('pay-button').addEventListener('click', async () => {
    document.getElementById('pay-button').disabled = true;
    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: { return_url: window.location.origin + '/i/%s/confirmed' },
    });
    if (error) {
      document.getElementById('error-message').textContent = error.message;
      document.getElementById('pay-button').disabled = false;
    }
  });
</script>`, inv.Total(), stripePK, clientSecret, inv.ViewToken)
	}

	if inv.Notes != "" {
		fmt.Fprintf(w, `<p>Notes: %s</p>`, inv.Notes)
	}

	fmt.Fprintf(w, `</body></html>`)
}

func (h *Handler) renderPaidPage(w http.ResponseWriter, inv *Invoice) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Invoice %s — Paid</title></head><body>`, inv.Number)
	fmt.Fprintf(w, `<h1>Invoice %s</h1>`, inv.Number)
	fmt.Fprintf(w, `<p style="color:green;font-size:24px"><strong>PAID</strong></p>`)
	if inv.PaidAt != nil {
		fmt.Fprintf(w, `<p>Payment received: %s</p>`, inv.PaidAt.Format("January 2, 2006"))
	}
	fmt.Fprintf(w, `<p>Total: $%.2f</p>`, inv.Total())
	fmt.Fprintf(w, `<p>Thank you for your payment!</p>`)
	fmt.Fprintf(w, `</body></html>`)
}

// PaymentConfirmed renders the thank-you page after Stripe redirect.
func (h *Handler) PaymentConfirmed(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	inv, err := h.store.GetByToken(r.Context(), token)
	if err != nil || inv == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Payment Received</title></head><body>`)
	fmt.Fprintf(w, `<h1>Thank You!</h1>`)
	fmt.Fprintf(w, `<p>Your payment for invoice <strong>%s</strong> has been received.</p>`, inv.Number)
	fmt.Fprintf(w, `<p>Amount: <strong>$%.2f</strong></p>`, inv.Total())
	fmt.Fprintf(w, `<p>You will receive a receipt via email.</p>`)
	fmt.Fprintf(w, `</body></html>`)
}
