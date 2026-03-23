package auth

import (
	"context"
	"net/http"
	"time"
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

func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userKey).(int64)
	return id
}
