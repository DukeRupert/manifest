package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"fireflysoftware.dev/manifest/internal/auth"
	"fireflysoftware.dev/manifest/internal/client"
	"fireflysoftware.dev/manifest/internal/db"
	"fireflysoftware.dev/manifest/internal/invoice"
	"fireflysoftware.dev/manifest/internal/server"
	"fireflysoftware.dev/manifest/internal/settings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		runSeed()
		return
	}
	runServer()
}

func runServer() {
	dsn := buildDSN()
	pool, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	authStore := auth.NewSessionStore(pool)
	clientStore := client.NewStore(pool)
	clientHandler := client.NewHandler(clientStore)
	settingsStore := settings.NewStore(pool)
	settingsHandler := settings.NewHandler(settingsStore)
	invoiceStore := invoice.NewStore(pool)
	invoiceHandler := invoice.NewHandler(invoiceStore, clientStore, settingsStore)

	handler := server.New(authStore, clientHandler, invoiceHandler, settingsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("manifest listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func buildDSN() string {
	host := envOr("DB_HOST", "localhost")
	port := envOr("DB_PORT", "5432")
	user := envOr("DB_USER", "manifest")
	pass := envOr("DB_PASSWORD", "changeme")
	name := envOr("DB_NAME", "manifest")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
