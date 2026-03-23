# Manifest — Phase 1: Foundation

## Overview

Single-tenant invoicing and billing app for Firefly Software. This phase establishes the project scaffold, infrastructure config, database foundation, and session-based authentication. No registration flow — a single admin user is seeded via migration or CLI command.

---

## Directory Structure

```
manifest/
├── cmd/
│   └── manifest/
│       └── main.go               # Entry point
├── internal/
│   ├── auth/
│   │   ├── handler.go            # Login/logout HTTP handlers
│   │   ├── middleware.go         # Session auth middleware
│   │   └── session.go            # Session creation/validation
│   ├── client/
│   │   ├── handler.go            # Client CRUD HTTP handlers
│   │   ├── model.go              # Client struct
│   │   └── store.go              # DB queries (sqlc-generated or hand-written)
│   ├── db/
│   │   └── db.go                 # DB connection pool init
│   └── server/
│       └── server.go             # HTTP server setup, route registration
├── migrations/
│   ├── 00001_create_users.sql
│   ├── 00002_create_sessions.sql
│   └── 00003_create_clients.sql
├── templates/                    # templ files (UI — not in scope for this doc)
├── static/                       # CSS, JS assets
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

---

## Docker Compose

```yaml
# docker-compose.yml
services:
  app:
    build: .
    restart: unless-stopped
    env_file: .env
    ports:
      - "127.0.0.1:8090:8080"   # bind to localhost only — Caddy proxies from host
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

> Port `8090` is the host-side port — pick any free port that doesn't conflict with your other services. The app always listens on `8080` inside the container.

Add this block to your host Caddyfile:

```caddy
manifest.fireflysoftware.dev {
    reverse_proxy localhost:8090
}
```

---

## Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o manifest ./cmd/manifest

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/manifest .
EXPOSE 8080
CMD ["./manifest"]
```

---

## Environment Variables

```bash
# .env.example
APP_ENV=production
APP_SECRET=change-me-32-char-random-string

DB_HOST=db
DB_PORT=5432
DB_USER=manifest
DB_PASSWORD=changeme
DB_NAME=manifest

PORT=8080
```

---

## Database Migrations (goose)

### 00001_create_users.sql

```sql
-- +goose Up
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,             -- bcrypt hash
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- No seed here — run `manifest seed` to create the admin user interactively.

-- +goose Down
DROP TABLE users;
```

---

## Seed Command (`manifest seed`)

The `seed` subcommand creates the single admin user interactively. Run once after the first migration.

```
$ manifest seed
Email: logan@fireflysoftware.dev
Password: (hidden)
Confirm password: (hidden)
✓ Admin user created.
```

### cmd/manifest/main.go — subcommand dispatch

```go
func main() {
    if len(os.Args) > 1 && os.Args[1] == "seed" {
        runSeed()
        return
    }
    runServer()
}
```

### cmd/manifest/seed.go

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "golang.org/x/crypto/bcrypt"
    "golang.org/x/term"
    "manifest/internal/db"
)

func runSeed() {
    pool, err := db.Connect(os.Getenv("DATABASE_URL"))
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

    fmt.Println("✓ Admin user created.")
}
```

> The `ON CONFLICT ... DO UPDATE` means re-running `manifest seed` with the same email resets the password — useful if you're locked out.

Add `golang.org/x/term` to dependencies alongside `pgx/v5`, `bcrypt`, and `goose`.

---

### 00002_create_sessions.sql

```sql
-- +goose Up
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,           -- random 32-byte hex token
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- +goose Down
DROP TABLE sessions;
```

---

### 00003_create_clients.sql

```sql
-- +goose Up
CREATE TABLE clients (
    id               BIGSERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,  -- used in invoice numbering, e.g. "ACME"
    email            TEXT,
    phone            TEXT,
    billing_address  TEXT,                  -- freeform for now
    notes            TEXT,
    archived_at      TIMESTAMPTZ,           -- soft delete
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clients_slug ON clients(slug);

-- +goose Down
DROP TABLE clients;
```

---

## Auth Design

### Session Flow

1. User POSTs credentials to `POST /login`
2. Server validates email/password (bcrypt compare)
3. On success: generate a 32-byte crypto-random token, insert into `sessions` with a 30-day expiry
4. Set `manifest_session` cookie (HttpOnly, Secure, SameSite=Lax)
5. Redirect to `/`
6. On every protected request: middleware reads cookie, queries `sessions` table, attaches user to context
7. `POST /logout` deletes session row and clears cookie

### session.go

```go
package auth

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "time"
)

const SessionCookieName = "manifest_session"
const SessionDuration = 30 * 24 * time.Hour

type Session struct {
    ID        string
    UserID    int64
    ExpiresAt time.Time
}

func GenerateSessionID() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

### middleware.go

```go
package auth

import (
    "context"
    "net/http"
)

type contextKey string

const userKey contextKey = "user"

func (s *SessionStore) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie(SessionCookieName)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        session, err := s.GetSession(r.Context(), cookie.Value)
        if err != nil || session == nil || session.ExpiresAt.Before(time.Now()) {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        ctx := context.WithValue(r.Context(), userKey, session.UserID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## Client Model

```go
package client

import "time"

type Client struct {
    ID             int64
    Name           string
    Slug           string      // uppercase, e.g. "ACME" — used in invoice numbers
    Email          string
    Phone          string
    BillingAddress string
    Notes          string
    ArchivedAt     *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### Slug Generation Rules

- Derived from client name on creation: `"Acme Corp"` → `"ACME"`
- Strip non-alphanumeric, uppercase, truncate to 8 chars
- If collision exists, append a number: `ACME2`
- User can override slug manually on the client form
- Slug is immutable after any invoice has been created for that client

---

## Route Registration (Phase 1)

```
GET  /login              → auth.ShowLogin
POST /login              → auth.HandleLogin
POST /logout             → auth.HandleLogout   [authenticated]

GET  /                   → dashboard (stub for now)  [authenticated]

GET  /clients            → client.List          [authenticated]
GET  /clients/new        → client.New           [authenticated]
POST /clients            → client.Create        [authenticated]
GET  /clients/{id}       → client.Show          [authenticated]
GET  /clients/{id}/edit  → client.Edit          [authenticated]
POST /clients/{id}       → client.Update        [authenticated]
POST /clients/{id}/archive → client.Archive     [authenticated]
```

> Use `POST` for updates/deletes (htmx-friendly, no JS method override needed).

---

## Server Init (server.go)

```go
package server

import (
    "net/http"
    "manifest/internal/auth"
    "manifest/internal/client"
)

func New(authStore *auth.SessionStore, clientStore *client.Store) http.Handler {
    mux := http.NewServeMux()

    // Public
    mux.HandleFunc("GET /login", authStore.ShowLogin)
    mux.HandleFunc("POST /login", authStore.HandleLogin)

    // Protected
    protected := http.NewServeMux()
    protected.HandleFunc("POST /logout", authStore.HandleLogout)
    protected.HandleFunc("GET /clients", clientStore.List)
    // ... etc

    mux.Handle("/", authStore.Middleware(protected))

    return mux
}
```

---

## main.go Sketch

```go
package main

import (
    "log"
    "net/http"
    "os"

    "manifest/internal/auth"
    "manifest/internal/client"
    "manifest/internal/db"
    "manifest/internal/server"
)

func main() {
    pool, err := db.Connect(os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatalf("db connect: %v", err)
    }

    authStore := auth.NewSessionStore(pool)
    clientStore := client.NewStore(pool)

    handler := server.New(authStore, clientStore)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("manifest listening on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}
```

---

## Phase 1 Checklist

- [ ] `go mod init fireflysoftware.dev/manifest`
- [ ] Add dependencies: `pgx/v5`, `bcrypt`, `goose`, `golang.org/x/term`
- [ ] Write and run migrations 00001–00003
- [ ] Implement `manifest seed` subcommand — creates/resets admin user interactively
- [ ] Implement `auth` package (session store, middleware, handlers)
- [ ] Implement `client` package (store, handlers)
- [ ] Register routes in `server.go`
- [ ] Docker Compose up — confirm DB connection and login flow
- [ ] Add Caddy block to host Caddyfile, verify HTTPS
- [ ] Run `manifest seed`, test login, create client, verify slug generation, archive client

---

## What's Not In This Phase

- Invoice creation (Phase 2)
- Stripe (Phase 3)
- Expenses (Phase 4)
- Reporting (Phase 5)
- PDF generation
- Settings / tax rate (deferred to Phase 2 — needed before first invoice)
