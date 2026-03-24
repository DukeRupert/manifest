package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"fireflysoftware.dev/manifest/internal/db"
)

func runSeed() {
	dsn := buildDSN()
	pool, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Org setup
	fmt.Print("Org name [Firefly Software]: ")
	var orgName string
	fmt.Scanln(&orgName)
	orgName = strings.TrimSpace(orgName)
	if orgName == "" {
		orgName = "Firefly Software"
	}

	// Derive slug from org name
	slug := strings.ToLower(orgName)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Find or create org
	var orgID string
	err = pool.QueryRow(ctx,
		`INSERT INTO orgs (name, slug)
		 VALUES ($1, $2)
		 ON CONFLICT (slug) DO UPDATE SET name = $1
		 RETURNING id`,
		orgName, slug,
	).Scan(&orgID)
	if err != nil {
		log.Fatalf("org setup: %v", err)
	}
	fmt.Printf("Org: %s (%s)\n", orgName, slug)

	// User setup
	fmt.Print("Name: ")
	var name string
	fmt.Scanln(&name)

	fmt.Print("Email: ")
	var email string
	fmt.Scanln(&email)

	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Fatalf("read password: %v", err)
	}

	fmt.Print("Confirm password: ")
	pw2, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Fatalf("read password: %v", err)
	}

	if string(pw) != string(pw2) {
		log.Fatal("passwords do not match")
	}

	hash, err := bcrypt.GenerateFromPassword(pw, 12)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO users (email, password_hash, org_id, name, role)
		 VALUES ($1, $2, $3, $4, 'admin')
		 ON CONFLICT ON CONSTRAINT users_org_email_unique
		 DO UPDATE SET password_hash = $2, name = $4, updated_at = NOW()`,
		email, string(hash), orgID, name,
	)
	if err != nil {
		log.Fatalf("insert user: %v", err)
	}

	fmt.Println("✓ Admin user created.")
}
