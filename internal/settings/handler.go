package settings

import (
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

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	st, err := h.store.Get(r.Context())
	if err != nil {
		http.Error(w, "failed to load settings", http.StatusInternalServerError)
		return
	}

	saved := r.URL.Query().Get("saved") == "1"
	v := &templates.SettingsView{
		BusinessName:    st.BusinessName,
		BusinessAddress: st.BusinessAddress,
		BusinessEmail:   st.BusinessEmail,
		DefaultTaxRate:  st.DefaultTaxRate,
		StripePK:        st.StripePK,
	}
	templates.SettingsShow(v, saved).Render(r.Context(), w)
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

	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}
