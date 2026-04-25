package middleware

import (
	"net/http"
	"res-cms-go/internal/session"

	"github.com/google/uuid"
)

// Auth middleware checks if user is authenticated
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("rescms_session")
		if err != nil {
			http.Redirect(w, r, "/access/login", http.StatusFound)
			return
		}

		// Decode session
		store := session.Get()
		sess, err := store.Decode(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/access/login", http.StatusFound)
			return
		}

		// Get session from store (validates expiration)
		session, ok := store.Get(sess.ID)
		if !ok {
			http.Redirect(w, r, "/access/login", http.StatusFound)
			return
		}

		// Add user info to request context
		ctx := WithUser(r.Context(), session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminAuth middleware checks if user is admin
func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("rescms_session")
		if err != nil {
			http.Redirect(w, r, "/access/login", http.StatusFound)
			return
		}

		// Decode session
		store := session.Get()
		sess, err := store.Decode(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/access/login", http.StatusFound)
			return
		}

		// Get session from store
		session, ok := store.Get(sess.ID)
		if !ok || !session.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Add user info to request context
		ctx := WithUser(r.Context(), session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireUser returns user from context
func RequireUser(r *http.Request) *session.Session {
	user, ok := r.Context().Value(userKey).(*session.Session)
	if !ok {
		return nil
	}
	return user
}

// OptionalUser returns user from context if authenticated
func OptionalUser(r *http.Request) *session.Session {
	user, _ := r.Context().Value(userKey).(*session.Session)
	return user
}

// GenerateFlashCookie generates a flash cookie
func GenerateFlashCookie(w http.ResponseWriter, msg string) {
	flash := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "rescms_flash",
		Value:    flash + ":" + msg,
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
	})
}

// GetFlashFromRequest reads and clears flash message from cookie
func GetFlashFromRequest(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("rescms_flash")
	if err != nil {
		return ""
	}

	// Capture the original value before clearing
	originalValue := cookie.Value

	// Clear the flash cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "rescms_flash",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Extract message after the first ":"
	for i := 0; i < len(originalValue); i++ {
		if originalValue[i] == ':' {
			return originalValue[i+1:]
		}
	}
	return ""
}
