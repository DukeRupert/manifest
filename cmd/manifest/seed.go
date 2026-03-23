package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (email, password) VALUES ($1, $2)
		 ON CONFLICT (email) DO UPDATE SET password = $2, updated_at = NOW()`,
		email, string(hash),
	)
	if err != nil {
		log.Fatalf("insert user: %v", err)
	}

	fmt.Println("Admin user created.")
}
