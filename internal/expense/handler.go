package expense

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// --- Category Handlers ---

func (h *Handler) CategoryList(w http.ResponseWriter, r *http.Request) {
	cats, err := h.store.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "failed to list categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Expense Categories (%d)</h1>`, len(cats))
	fmt.Fprintf(w, `<a href="/expenses/categories/new">New Category</a>`)
	for _, c := range cats {
		fmt.Fprintf(w, `<div>
  <form method="POST" action="/expenses/categories/%d" style="display:inline">
    <input type="text" name="name" value="%s" required>
    <button type="submit">Save</button>
  </form>
  <form method="POST" action="/expenses/categories/delete/%d" style="display:inline">
    <button type="submit">Delete</button>
  </form>
</div>`, c.ID, c.Name, c.ID)
	}
}

func (h *Handler) CategoryNew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<h1>New Category</h1>
<form method="POST" action="/expenses/categories">
  <label>Name<br><input type="text" name="name" required></label><br>
  <button type="submit">Create</button>
</form>`))
}

func (h *Handler) CategoryCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if _, err := h.store.CreateCategory(r.Context(), name); err != nil {
		http.Error(w, fmt.Sprintf("failed to create category: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/expenses/categories", http.StatusSeeOther)
}

func (h *Handler) CategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateCategory(r.Context(), id, name); err != nil {
		http.Error(w, fmt.Sprintf("failed to update category: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/expenses/categories", http.StatusSeeOther)
}

func (h *Handler) CategoryDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.store.DeleteCategory(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete: %v", err), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/expenses/categories", http.StatusSeeOther)
}

// --- Expense Handlers ---

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.From = &t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.To = &t
		}
	}
	if catID := r.URL.Query().Get("category_id"); catID != "" {
		if id, err := strconv.ParseInt(catID, 10, 64); err == nil {
			filter.CategoryID = &id
		}
	}

	expenses, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to list expenses", http.StatusInternalServerError)
		return
	}

	cats, _ := h.store.ListCategories(r.Context())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Expenses (%d)</h1>`, len(expenses))

	// Filter form
	fromVal, toVal := "", ""
	if filter.From != nil {
		fromVal = filter.From.Format("2006-01-02")
	}
	if filter.To != nil {
		toVal = filter.To.Format("2006-01-02")
	}
	fmt.Fprintf(w, `<form method="GET" action="/expenses">
  <label>From <input type="date" name="from" value="%s"></label>
  <label>To <input type="date" name="to" value="%s"></label>
  <label>Category <select name="category_id"><option value="">All</option>`, fromVal, toVal)
	for _, c := range cats {
		selected := ""
		if filter.CategoryID != nil && *filter.CategoryID == c.ID {
			selected = " selected"
		}
		fmt.Fprintf(w, `<option value="%d"%s>%s</option>`, c.ID, selected, c.Name)
	}
	fmt.Fprintf(w, `</select></label>
  <button type="submit">Filter</button>
</form>`)

	fmt.Fprintf(w, `<a href="/expenses/new">New Expense</a> | <a href="/expenses/categories">Categories</a>`)

	var total float64
	fmt.Fprintf(w, `<table><tr><th>Date</th><th>Vendor</th><th>Category</th><th>Amount</th><th>Notes</th><th></th></tr>`)
	for _, e := range expenses {
		total += e.Amount
		fmt.Fprintf(w, `<tr><td>%s</td><td>%s</td><td>%s</td><td>$%.2f</td><td>%s</td><td><a href="/expenses/%d/edit">Edit</a></td></tr>`,
			e.Date.Format("2006-01-02"), e.Vendor, e.Category.Name, e.Amount, e.Notes, e.ID)
	}
	fmt.Fprintf(w, `</table>`)
	fmt.Fprintf(w, `<p><strong>Total: $%.2f</strong></p>`, total)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	cats, _ := h.store.ListCategories(r.Context())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>New Expense</h1>
<form method="POST" action="/expenses">
  <label>Date<br><input type="date" name="date" value="%s" required></label><br>
  <label>Vendor<br><input type="text" name="vendor" required></label><br>
  <label>Amount<br><input type="number" step="0.01" name="amount" required></label><br>
  <label>Category<br><select name="category_id" required>`, time.Now().Format("2006-01-02"))
	for _, c := range cats {
		fmt.Fprintf(w, `<option value="%d">%s</option>`, c.ID, c.Name)
	}
	fmt.Fprintf(w, `</select></label><br>
  <label>Notes<br><textarea name="notes"></textarea></label><br>
  <button type="submit">Create</button>
</form>`)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	date, _ := time.Parse("2006-01-02", r.FormValue("date"))

	e := &Expense{
		CategoryID: catID,
		Vendor:     r.FormValue("vendor"),
		Amount:     amount,
		Notes:      r.FormValue("notes"),
		Date:       date,
	}

	if err := h.store.Create(r.Context(), e); err != nil {
		http.Error(w, fmt.Sprintf("failed to create expense: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	e, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cats, _ := h.store.ListCategories(r.Context())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<h1>Edit Expense</h1>
<form method="POST" action="/expenses/%d">
  <label>Date<br><input type="date" name="date" value="%s" required></label><br>
  <label>Vendor<br><input type="text" name="vendor" value="%s" required></label><br>
  <label>Amount<br><input type="number" step="0.01" name="amount" value="%.2f" required></label><br>
  <label>Category<br><select name="category_id" required>`, e.ID, e.Date.Format("2006-01-02"), e.Vendor, e.Amount)
	for _, c := range cats {
		selected := ""
		if c.ID == e.CategoryID {
			selected = " selected"
		}
		fmt.Fprintf(w, `<option value="%d"%s>%s</option>`, c.ID, selected, c.Name)
	}
	fmt.Fprintf(w, `</select></label><br>
  <label>Notes<br><textarea name="notes">%s</textarea></label><br>
  <button type="submit">Save</button>
</form>
<form method="POST" action="/expenses/delete/%d">
  <button type="submit">Delete</button>
</form>`, e.Notes, e.ID)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	date, _ := time.Parse("2006-01-02", r.FormValue("date"))

	e := &Expense{
		ID:         id,
		CategoryID: catID,
		Vendor:     r.FormValue("vendor"),
		Amount:     amount,
		Notes:      r.FormValue("notes"),
		Date:       date,
	}

	if err := h.store.Update(r.Context(), e); err != nil {
		http.Error(w, fmt.Sprintf("failed to update: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}
