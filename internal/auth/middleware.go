package auth

import (
	"context"
	"net/http"
	"time"
)

type contextKey string

const userKey contextKey = "user"
const orgKey contextKey = "org"

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
		ctx = context.WithValue(ctx, orgKey, session.OrgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserID returns the authenticated user's UUID from the request context.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userKey).(string)
	return id
}

// OrgID returns the authenticated user's org UUID from the request context.
func OrgID(ctx context.Context) string {
	id, _ := ctx.Value(orgKey).(string)
	return id
}
