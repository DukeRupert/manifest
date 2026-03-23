package client

import (
	"fmt"
	"net/http"
	"strconv"

	"fireflysoftware.dev/manifest/templates"
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
	views := make([]templates.ClientView, len(clients))
	for i, c := range clients {
		views[i] = toView(&c)
	}
	templates.ClientsList(views).Render(r.Context(), w)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	templates.ClientsNew().Render(r.Context(), w)
}

func toView(c *Client) templates.ClientView {
	return templates.ClientView{
		ID:             c.ID,
		Name:           c.Name,
		Slug:           c.Slug,
		Email:          c.Email,
		Phone:          c.Phone,
		BillingAddress: c.BillingAddress,
		Notes:          c.Notes,
		ArchivedAt:     c.ArchivedAt,
	}
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

	v := toView(c)
	templates.ClientsShow(&v).Render(r.Context(), w)
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

	v := toView(c)
	templates.ClientsEdit(&v).Render(r.Context(), w)
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
