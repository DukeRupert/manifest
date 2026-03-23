package auth

import (
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (s *SessionStore) ShowLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// TODO: render templ login template
	w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Login — Manifest</title></head>
<body>
<h1>Login</h1>
<form method="POST" action="/login">
  <label>Email<br><input type="email" name="email" required></label><br>
  <label>Password<br><input type="password" name="password" required></label><br>
  <button type="submit">Log In</button>
</form>
</body></html>`))
}

func (s *SessionStore) HandleLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var userID int64
	var hash string
	err := s.pool.QueryRow(r.Context(),
		`SELECT id, password FROM users WHERE email = $1`, email,
	).Scan(&userID, &hash)
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	sessionID, err := s.CreateSession(r.Context(), userID)
	if err != nil {
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionDuration),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *SessionStore) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		s.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
