package handlers

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"res-cms-go/internal/db"
	"res-cms-go/internal/models"
	"res-cms-go/internal/session"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) func() {
	dbFile := "test_auth.db"
	var err error
	db.DB, err = gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate schemas
	err = db.DB.AutoMigrate(&models.User{}, &models.SiteSetting{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Initialize session store
	session.Init([]string{"test-secret"})

	// Create a user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	user := models.User{
		Username: "testuser",
		Password: string(hashedPassword),
		Email:    "test@example.com",
		Status:   "activated",
		IsAdmin:  false,
	}
	db.DB.Create(&user)

	cleanup := func() {
		sqlDB, _ := db.DB.DB()
		sqlDB.Close()
		os.Remove(dbFile)
		os.Remove(dbFile + "-shm")
		os.Remove(dbFile + "-wal")
	}

	return cleanup
}

func TestLoginHandler_Success(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "password123")

	req, err := http.NewRequest("POST", "/access/login", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LoginHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	if loc := rr.Header().Get("Location"); loc != "/manage" {
		t.Errorf("expected redirect to /manage, got %s", loc)
	}

	// Check if cookie is set
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "rescms_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("session cookie not set")
	}
}

func TestLoginHandler_Failure(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "wrongpassword")

	req, err := http.NewRequest("POST", "/access/login", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LoginHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	if loc := rr.Header().Get("Location"); loc != "/access/login" {
		t.Errorf("expected redirect to /access/login, got %s", loc)
	}
}

func TestLogoutHandler(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	// Create a dummy session
	store := session.Get()
	sess, _ := store.Create(1, "testuser", false)
	encoded, _ := store.Encode(sess)

	req, err := http.NewRequest("GET", "/access/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:  "rescms_session",
		Value: encoded,
	})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LogoutHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}

	// Check if cookie was cleared
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "rescms_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil || sessionCookie.Value != "" || sessionCookie.MaxAge != -1 {
		t.Errorf("session cookie not correctly cleared")
	}

	// Verify session is removed from store
	_, ok := store.Get(sess.ID)
	if ok {
		t.Errorf("session still exists in store after logout")
	}
}

// Dummy functions to satisfy templates and other dependencies if needed
func init() {
	// We might need to override log output during tests
	log.SetOutput(os.Stdout)
}

// We need a dummy renderTemplate for LoginFormHandler if we were to test it, 
// but LoginFormHandler renders HTML which is harder to test without template setup.
// For now we focus on logic handlers.
