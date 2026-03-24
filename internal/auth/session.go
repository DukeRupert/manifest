package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const SessionCookieName = "manifest_session"
const SessionDuration = 30 * 24 * time.Hour

type Session struct {
	ID        string // UUID
	UserID    string // UUID
	OrgID     string // UUID (joined from users)
	ExpiresAt time.Time
}

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

// GenerateToken creates a 32-byte crypto-random hex token for the session cookie.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken computes the SHA-256 hash of a raw token string for DB storage.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (s *SessionStore) CreateSession(ctx context.Context, userID string) (string, error) {
	rawToken, err := GenerateToken()
	if err != nil {
		return "", err
	}
	tokenHash := HashToken(rawToken)
	expiresAt := time.Now().Add(SessionDuration)
	_, err = s.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		rawToken, userID, tokenHash, expiresAt,
	)
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *SessionStore) GetSession(ctx context.Context, rawToken string) (*Session, error) {
	tokenHash := HashToken(rawToken)
	var sess Session
	err := s.pool.QueryRow(ctx,
		`SELECT s.uuid, u.uuid, u.org_id, s.expires_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1`,
		tokenHash,
	).Scan(&sess.ID, &sess.UserID, &sess.OrgID, &sess.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *SessionStore) DeleteSession(ctx context.Context, rawToken string) error {
	tokenHash := HashToken(rawToken)
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
