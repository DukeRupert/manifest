package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const SessionCookieName = "manifest_session"
const SessionDuration = 30 * 24 * time.Hour

type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
}

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *SessionStore) CreateSession(ctx context.Context, userID int64) (string, error) {
	id, err := GenerateSessionID()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(SessionDuration)
	_, err = s.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		id, userID, expiresAt,
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *SessionStore) GetSession(ctx context.Context, id string) (*Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, expires_at FROM sessions WHERE id = $1`,
		id,
	).Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *SessionStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}
