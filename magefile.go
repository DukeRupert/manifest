//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Run

// Templ generates Go code from .templ files.
func Templ() error {
	fmt.Println("Generating templ...")
	return sh.Run("templ", "generate")
}

// CSS compiles Tailwind CSS.
func CSS() error {
	fmt.Println("Building CSS...")
	return sh.Run("./tailwindcss", "-i", "static/css/input.css", "-o", "static/css/app.css", "--minify")
}

// Build generates templ, compiles CSS, and builds the manifest binary.
func Build() error {
	mg.Deps(Templ, CSS)
	fmt.Println("Building manifest...")
	return sh.Run("go", "build", "-o", "manifest", "./cmd/manifest")
}

// Run builds and starts the server using the local Docker DB.
func Run() error {
	mg.Deps(Build)
	setLocalDBEnv()
	fmt.Println("Starting manifest...")
	return sh.RunV("./manifest")
}

// Seed runs the interactive admin user seed command.
func Seed() error {
	mg.Deps(Build)
	setLocalDBEnv()
	return sh.RunV("./manifest", "seed")
}

// setLocalDBEnv sets DB_PORT to the exposed Docker port if not already set.
func setLocalDBEnv() {
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", "5433")
	}
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
