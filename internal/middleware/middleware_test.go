package middleware

import (
	"net/http"
	"net/http/httptest"
	"res-cms-go/internal/session"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	// Initialize session store
	session.Init([]string{"test-secret"})
	store := session.Get()

	// Create a session
	sess, _ := store.Create(1, "testuser", false)
	encoded, _ := store.Encode(sess)

	// Create a handler that uses the middleware
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := RequireUser(r)
		if user == nil {
			t.Error("User not found in context")
		}
		if user.Username != "testuser" {
			t.Errorf("Expected username testuser, got %s", user.Username)
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := Auth(nextHandler)

	// 1. Test without cookie
	req, _ := http.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Errorf("Expected redirect, got %d", rr.Code)
	}

	// 2. Test with valid cookie
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "rescms_session",
		Value: encoded,
	})
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}
}

func TestAdminAuthMiddleware(t *testing.T) {
	// Initialize session store
	session.Init([]string{"test-secret"})
	store := session.Get()

	// 1. Test with non-admin session
	sess, _ := store.Create(1, "testuser", false)
	encoded, _ := store.Encode(sess)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := AdminAuth(nextHandler)

	req, _ := http.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{
		Name:  "rescms_session",
		Value: encoded,
	})
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for non-admin, got %d", rr.Code)
	}

	// 2. Test with admin session
	adminSess, _ := store.Create(2, "admin", true)
	adminEncoded, _ := store.Encode(adminSess)

	req, _ = http.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{
		Name:  "rescms_session",
		Value: adminEncoded,
	})
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for admin, got %d", rr.Code)
	}
}
