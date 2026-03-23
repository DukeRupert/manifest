//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Run

// Build compiles the manifest binary.
func Build() error {
	fmt.Println("Building manifest...")
	return sh.Run("go", "build", "-o", "manifest", "./cmd/manifest")
}

// Run builds and starts the server.
func Run() error {
	mg.Deps(Build)
	fmt.Println("Starting manifest...")
	return sh.RunV("./manifest")
}

// Seed runs the interactive admin user seed command.
func Seed() error {
	mg.Deps(Build)
	return sh.RunV("./manifest", "seed")
}

// Migrate runs all pending goose migrations.
func Migrate() error {
	return sh.RunV("goose", "-dir", "migrations", "postgres", dsn(), "up")
}

// MigrateDown rolls back the last migration.
func MigrateDown() error {
	return sh.RunV("goose", "-dir", "migrations", "postgres", dsn(), "down")
}

// MigrateStatus shows current migration status.
func MigrateStatus() error {
	return sh.RunV("goose", "-dir", "migrations", "postgres", dsn(), "status")
}

// Vet runs go vet on all packages.
func Vet() error {
	return sh.RunV("go", "vet", "./...")
}

// Test runs all tests.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// Docker builds and starts the full Docker Compose stack.
func Docker() error {
	return sh.RunV("docker", "compose", "up", "--build", "-d")
}

// DockerDown stops and removes the Docker Compose stack.
func DockerDown() error {
	return sh.RunV("docker", "compose", "down")
}

// DockerLogs tails the app container logs.
func DockerLogs() error {
	return sh.RunV("docker", "compose", "logs", "-f", "app")
}

// Clean removes the compiled binary.
func Clean() error {
	fmt.Println("Cleaning...")
	return os.Remove("manifest")
}

func dsn() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	host := envOr("DB_HOST", "localhost")
	port := envOr("DB_PORT", "5433")
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
