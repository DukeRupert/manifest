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
	"fireflysoftware.dev/manifest/templates"
)

// PaymentCreator creates or retrieves a Stripe PaymentIntent client secret.
// Returns empty string if Stripe is not configured.
type PaymentCreator func(ctx context.Context, inv *Invoice) (clientSecret string, err error)

type Handler struct {
	store         *Store
	clientStore   *clientpkg.Store
	settingsStore *settings.Store
	createPayment PaymentCreator
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
	statusFilter := Status(r.URL.Query().Get("status"))
	items, err := h.store.List(r.Context(), statusFilter)
	if err != nil {
		http.Error(w, "failed to list invoices", http.StatusInternalServerError)
		return
	}

	views := make([]templates.InvoiceListItemView, len(items))
	for i, item := range items {
		views[i] = templates.InvoiceListItemView{
			ID:         item.ID,
			Number:     item.Number,
			ClientName: item.ClientName,
			Status:     templates.InvoiceStatus(item.Status),
			DueDate:    item.DueDate,
			IssuedAt:   item.IssuedAt,
			Total:      item.Total,
		}
	}
	templates.InvoicesList(views, string(statusFilter)).Render(r.Context(), w)
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

	clientViews := make([]templates.ClientView, len(clients))
	for i, c := range clients {
		clientViews[i] = templates.ClientView{ID: c.ID, Name: c.Name, Slug: c.Slug}
	}

	templates.InvoicesNew(clientViews, st.DefaultTaxRate).Render(r.Context(), w)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid client", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
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

	v := toInvoiceView(inv)
	templates.InvoicesShow(&v).Render(r.Context(), w)
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

	clients, err := h.clientStore.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list clients", http.StatusInternalServerError)
		return
	}

	clientViews := make([]templates.ClientView, len(clients))
	for i, c := range clients {
		clientViews[i] = templates.ClientView{ID: c.ID, Name: c.Name, Slug: c.Slug}
	}

	v := toInvoiceView(inv)
	templates.InvoicesEdit(&v, clientViews).Render(r.Context(), w)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
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
		v := toInvoiceView(inv)
		st, _ := h.settingsStore.Get(r.Context())
		businessName := ""
		if st != nil {
			businessName = st.BusinessName
		}
		templates.InvoicePaid(&v, businessName).Render(r.Context(), w)
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
			fmt.Printf("payment setup error: %v\n", err)
		}
	}

	v := toInvoiceView(inv)
	pv := &templates.PublicInvoiceView{
		Invoice:      v,
		StripePK:     stripePK,
		ClientSecret: clientSecret,
	}
	if st != nil {
		pv.BusinessName = st.BusinessName
		pv.BusinessAddress = st.BusinessAddress
		pv.BusinessEmail = st.BusinessEmail
	}
	pv.ClientAddress = inv.Client.BillingAddress

	templates.InvoicePublicView(pv).Render(r.Context(), w)
}

// PaymentConfirmed renders the thank-you page after Stripe redirect.
func (h *Handler) PaymentConfirmed(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	inv, err := h.store.GetByToken(r.Context(), token)
	if err != nil || inv == nil {
		http.NotFound(w, r)
		return
	}

	v := toInvoiceView(inv)
	templates.InvoicePaymentConfirmed(&v).Render(r.Context(), w)
}

func toInvoiceView(inv *Invoice) templates.InvoiceView {
	v := templates.InvoiceView{
		ID:         inv.ID,
		Number:     inv.Number,
		ClientID:   inv.ClientID,
		ClientName: inv.Client.Name,
		ClientSlug: inv.Client.Slug,
		Status:     templates.InvoiceStatus(inv.Status),
		TaxRate:    inv.TaxRate,
		Notes:      inv.Notes,
		DueDate:    inv.DueDate,
		IssuedAt:   inv.IssuedAt,
		PaidAt:     inv.PaidAt,
		ViewToken:  inv.ViewToken,
	}
	for _, li := range inv.LineItems {
		v.LineItems = append(v.LineItems, templates.LineItemView{
			ID:          li.ID,
			Description: li.Description,
			Quantity:    li.Quantity,
			UnitPrice:   li.UnitPrice,
		})
	}
	return v
}
