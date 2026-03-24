package expense

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fireflysoftware.dev/manifest/templates"
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
	views := toCategoryViews(cats)
	templates.ExpensesCategoryList(views).Render(r.Context(), w)
}

func (h *Handler) CategoryNew(w http.ResponseWriter, r *http.Request) {
	templates.ExpensesCategoryNew().Render(r.Context(), w)
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
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *Handler) CategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // UUID
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateCategory(r.Context(), id, name); err != nil {
		http.Error(w, fmt.Sprintf("failed to update category: %v", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *Handler) CategoryDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // UUID
	if err := h.store.DeleteCategory(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete: %v", err), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

// --- Expense Handlers ---

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{}
	filterFrom, filterTo := "", ""
	var filterCat string

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.From = &t
			filterFrom = from
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.To = &t
			filterTo = to
		}
	}
	if catID := r.URL.Query().Get("category_id"); catID != "" {
		filter.CategoryID = &catID
		filterCat = catID
	}

	expenses, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to list expenses", http.StatusInternalServerError)
		return
	}

	cats, _ := h.store.ListCategories(r.Context())

	var total float64
	views := make([]templates.ExpenseView, len(expenses))
	for i, e := range expenses {
		total += e.Amount
		views[i] = toExpenseView(&e)
	}

	data := templates.ExpenseListData{
		Expenses:   views,
		Categories: toCategoryViews(cats),
		FilterFrom: filterFrom,
		FilterTo:   filterTo,
		FilterCat:  filterCat,
		Total:      total,
	}
	templates.ExpensesList(data).Render(r.Context(), w)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	cats, _ := h.store.ListCategories(r.Context())
	templates.ExpensesNew(toCategoryViews(cats), time.Now().Format("2006-01-02")).Render(r.Context(), w)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	catID := r.FormValue("category_id") // UUID
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
	id := r.PathValue("id") // UUID

	e, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cats, _ := h.store.ListCategories(r.Context())
	v := toExpenseView(e)
	templates.ExpensesEdit(&v, toCategoryViews(cats)).Render(r.Context(), w)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // UUID

	catID := r.FormValue("category_id") // UUID
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
	id := r.PathValue("id") // UUID

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func toExpenseView(e *Expense) templates.ExpenseView {
	return templates.ExpenseView{
		ID:           e.ID,
		CategoryID:   e.CategoryID,
		CategoryName: e.Category.Name,
		Vendor:       e.Vendor,
		Amount:       e.Amount,
		Notes:        e.Notes,
		Date:         e.Date,
	}
}

func toCategoryViews(cats []Category) []templates.CategoryView {
	views := make([]templates.CategoryView, len(cats))
	for i, c := range cats {
		views[i] = templates.CategoryView{ID: c.ID, Name: c.Name}
	}
	return views
}
