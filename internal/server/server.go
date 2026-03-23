package server

import (
	"net/http"

	"fireflysoftware.dev/manifest/internal/auth"
	"fireflysoftware.dev/manifest/internal/client"
	"fireflysoftware.dev/manifest/internal/expense"
	"fireflysoftware.dev/manifest/internal/invoice"
	"fireflysoftware.dev/manifest/internal/payment"
	"fireflysoftware.dev/manifest/internal/reports"
	"fireflysoftware.dev/manifest/internal/settings"
)

func New(authStore *auth.SessionStore, clientHandler *client.Handler, invoiceHandler *invoice.Handler, settingsHandler *settings.Handler, webhookHandler *payment.WebhookHandler, expenseHandler *expense.Handler, reportsHandler *reports.Handler) http.Handler {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Public routes
	mux.HandleFunc("GET /login", authStore.ShowLogin)
	mux.HandleFunc("POST /login", authStore.HandleLogin)
	mux.HandleFunc("GET /i/{token}", invoiceHandler.PublicView)
	mux.HandleFunc("GET /i/{token}/confirmed", invoiceHandler.PaymentConfirmed)
	if webhookHandler != nil {
		mux.HandleFunc("POST /webhooks/stripe", webhookHandler.HandleStripeWebhook)
	}

	// Protected routes
	protected := http.NewServeMux()
	protected.HandleFunc("POST /logout", authStore.HandleLogout)

	// Dashboard stub
	protected.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<h1>Manifest</h1><p>Dashboard coming soon.</p>
<nav><a href="/clients">Clients</a> | <a href="/invoices">Invoices</a> | <a href="/expenses">Expenses</a> | <a href="/reports">Reports</a> | <a href="/settings">Settings</a></nav>`))
	})

	// Clients
	protected.HandleFunc("GET /clients", clientHandler.List)
	protected.HandleFunc("GET /clients/new", clientHandler.New)
	protected.HandleFunc("POST /clients", clientHandler.Create)
	protected.HandleFunc("GET /clients/{id}", clientHandler.Show)
	protected.HandleFunc("GET /clients/{id}/edit", clientHandler.Edit)
	protected.HandleFunc("POST /clients/{id}", clientHandler.Update)
	protected.HandleFunc("POST /clients/{id}/archive", clientHandler.Archive)

	// Settings
	protected.HandleFunc("GET /settings", settingsHandler.Show)
	protected.HandleFunc("POST /settings", settingsHandler.Update)

	// Invoices
	protected.HandleFunc("GET /invoices", invoiceHandler.List)
	protected.HandleFunc("GET /invoices/new", invoiceHandler.New)
	protected.HandleFunc("POST /invoices", invoiceHandler.Create)
	protected.HandleFunc("GET /invoices/{id}", invoiceHandler.Show)
	protected.HandleFunc("GET /invoices/{id}/edit", invoiceHandler.Edit)
	protected.HandleFunc("POST /invoices/{id}", invoiceHandler.Update)
	protected.HandleFunc("POST /invoices/{id}/send", invoiceHandler.Send)
	protected.HandleFunc("POST /invoices/{id}/void", invoiceHandler.Void)

	// Expense Categories
	protected.HandleFunc("GET /expenses/categories", expenseHandler.CategoryList)
	protected.HandleFunc("GET /expenses/categories/new", expenseHandler.CategoryNew)
	protected.HandleFunc("POST /expenses/categories", expenseHandler.CategoryCreate)
	protected.HandleFunc("POST /expenses/categories/{id}", expenseHandler.CategoryUpdate)
	protected.HandleFunc("POST /expenses/categories/delete/{id}", expenseHandler.CategoryDelete)

	// Expenses
	protected.HandleFunc("GET /expenses", expenseHandler.List)
	protected.HandleFunc("GET /expenses/new", expenseHandler.New)
	protected.HandleFunc("POST /expenses", expenseHandler.Create)
	protected.HandleFunc("GET /expenses/{id}/edit", expenseHandler.Edit)
	protected.HandleFunc("POST /expenses/{id}", expenseHandler.Update)
	protected.HandleFunc("POST /expenses/delete/{id}", expenseHandler.Delete)

	// Reports
	protected.HandleFunc("GET /reports", reportsHandler.Index)
	protected.HandleFunc("GET /reports/revenue", reportsHandler.Revenue)
	protected.HandleFunc("GET /reports/ar", reportsHandler.AR)
	protected.HandleFunc("GET /reports/pl", reportsHandler.PL)

	mux.Handle("/", authStore.Middleware(protected))

	return mux
}
