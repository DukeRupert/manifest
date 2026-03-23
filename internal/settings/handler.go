package settings

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

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	st, err := h.store.Get(r.Context())
	if err != nil {
		http.Error(w, "failed to load settings", http.StatusInternalServerError)
		return
	}

	// TODO: render templ template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Settings</h1>
<form method="POST" action="/settings">
  <label>Business Name<br><input type="text" name="business_name" value="%s"></label><br>
  <label>Business Address<br><textarea name="business_address">%s</textarea></label><br>
  <label>Business Email<br><input type="email" name="business_email" value="%s"></label><br>
  <label>Default Tax Rate (%%)<br><input type="number" step="0.01" name="default_tax_rate" value="%.2f"></label><br>
  <label>Stripe Publishable Key<br><input type="text" name="stripe_pk" value="%s"></label><br>
  <button type="submit">Save</button>
</form>`, st.BusinessName, st.BusinessAddress, st.BusinessEmail, st.DefaultTaxRate, st.StripePK)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	st, err := h.store.Get(r.Context())
	if err != nil {
		http.Error(w, "failed to load settings", http.StatusInternalServerError)
		return
	}

	st.BusinessName = r.FormValue("business_name")
	st.BusinessAddress = r.FormValue("business_address")
	st.BusinessEmail = r.FormValue("business_email")
	st.StripePK = r.FormValue("stripe_pk")

	if rate, err := strconv.ParseFloat(r.FormValue("default_tax_rate"), 64); err == nil {
		st.DefaultTaxRate = rate
	}

	if err := h.store.Update(r.Context(), st); err != nil {
		http.Error(w, "failed to save settings", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
