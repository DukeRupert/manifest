package client

import (
	"fmt"
	"net/http"
	"strconv"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	clients, err := h.store.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list clients", http.StatusInternalServerError)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>Clients (%d)</h1>", len(clients))
	for _, c := range clients {
		fmt.Fprintf(w, "<div><a href=\"/clients/%d\">%s</a> [%s]</div>", c.ID, c.Name, c.Slug)
	}
	fmt.Fprintf(w, `<a href="/clients/new">New Client</a>`)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<h1>New Client</h1>
<form method="POST" action="/clients">
  <label>Name<br><input type="text" name="name" required></label><br>
  <label>Slug (optional)<br><input type="text" name="slug"></label><br>
  <label>Email<br><input type="email" name="email"></label><br>
  <label>Phone<br><input type="text" name="phone"></label><br>
  <label>Billing Address<br><textarea name="billing_address"></textarea></label><br>
  <label>Notes<br><textarea name="notes"></textarea></label><br>
  <button type="submit">Create</button>
</form>`))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	c := &Client{
		Name:           r.FormValue("name"),
		Email:          r.FormValue("email"),
		Phone:          r.FormValue("phone"),
		BillingAddress: r.FormValue("billing_address"),
		Notes:          r.FormValue("notes"),
	}

	slug := r.FormValue("slug")
	if slug == "" {
		var err error
		slug, err = h.store.GenerateSlug(r.Context(), c.Name)
		if err != nil {
			http.Error(w, "slug generation failed", http.StatusInternalServerError)
			return
		}
	}
	c.Slug = slug

	if err := h.store.Create(r.Context(), c); err != nil {
		http.Error(w, "failed to create client", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%d", c.ID), http.StatusSeeOther)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	c, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>%s</h1><p>Slug: %s</p><p>Email: %s</p><p>Phone: %s</p>",
		c.Name, c.Slug, c.Email, c.Phone)
	fmt.Fprintf(w, `<a href="/clients/%d/edit">Edit</a>`, c.ID)
	fmt.Fprintf(w, `<form method="POST" action="/clients/%d/archive"><button type="submit">Archive</button></form>`, c.ID)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	c, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Edit %s</h1>
<form method="POST" action="/clients/%d">
  <label>Name<br><input type="text" name="name" value="%s" required></label><br>
  <label>Email<br><input type="email" name="email" value="%s"></label><br>
  <label>Phone<br><input type="text" name="phone" value="%s"></label><br>
  <label>Billing Address<br><textarea name="billing_address">%s</textarea></label><br>
  <label>Notes<br><textarea name="notes">%s</textarea></label><br>
  <button type="submit">Save</button>
</form>`, c.Name, c.ID, c.Name, c.Email, c.Phone, c.BillingAddress, c.Notes)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	c := &Client{
		ID:             id,
		Name:           r.FormValue("name"),
		Email:          r.FormValue("email"),
		Phone:          r.FormValue("phone"),
		BillingAddress: r.FormValue("billing_address"),
		Notes:          r.FormValue("notes"),
	}

	if err := h.store.Update(r.Context(), c); err != nil {
		http.Error(w, "failed to update client", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/clients/%d", id), http.StatusSeeOther)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.store.Archive(r.Context(), id); err != nil {
		http.Error(w, "failed to archive client", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}
