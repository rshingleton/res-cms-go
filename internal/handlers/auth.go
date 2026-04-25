package handlers

import (
	"log"
	"net/http"
	"res-cms-go/internal/db"
	"res-cms-go/internal/middleware"
	"res-cms-go/internal/models"
	"res-cms-go/internal/session"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"gorm.io/gorm"
)

// LoginFormHandler displays the login form
func LoginFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/access/login" {
		http.NotFound(w, r)
		return
	}

	// If already logged in, redirect to admin
	user := middleware.OptionalUser(r)
	if user != nil {
		http.Redirect(w, r, "/manage", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"res_blog_name": getBlogName(),
		"CSRFToken":     generateCSRFToken(),
	}

	if err := renderTemplate(w, r, "public/login.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// LoginHandler processes login credentials
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	if username == "" || password == "" {
		middleware.GenerateFlashCookie(w, "Username and password are required")
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	// Find user
	var user models.User
	if err := db.DB.Where("username = ? AND status = ?", username, "activated").First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			middleware.GenerateFlashCookie(w, "Invalid username or password")
			http.Redirect(w, r, "/access/login", http.StatusFound)
			return
		}
		log.Printf("Error finding user: %v", err)
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		middleware.GenerateFlashCookie(w, "Invalid username or password")
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	// Create session
	store := session.Get()
	sess, err := store.Create(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		middleware.GenerateFlashCookie(w, "Login failed")
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	// Encode and set cookie
	encoded, err := store.Encode(sess)
	if err != nil {
		log.Printf("Error encoding session: %v", err)
		middleware.GenerateFlashCookie(w, "Login failed")
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "rescms_session",
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		// Secure: true, // Enable in production
		MaxAge: 86400, // 24 hours
	})

	middleware.GenerateFlashCookie(w, "Logged in successfully")
	http.Redirect(w, r, "/manage", http.StatusFound)
}

// LogoutHandler logs out the user
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/access/logout" {
		http.NotFound(w, r)
		return
	}

	// Get session and destroy
	cookie, err := r.Cookie("rescms_session")
	if err == nil {
		store := session.Get()
		sess, err := store.Decode(cookie.Value)
		if err == nil {
			store.Destroy(sess.ID)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "rescms_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	middleware.GenerateFlashCookie(w, "Logged out successfully")
	http.Redirect(w, r, "/access/login", http.StatusFound)
}

// ProfileHandler displays and updates user profile
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	isPost := r.Method == http.MethodPost

	switch {
	case path == "/profile" && !isPost:
		profileView(w, r)
	case path == "/profile" && isPost:
		profileUpdate(w, r)
	default:
		http.NotFound(w, r)
	}
}

func profileView(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	var dbUser models.User
	if err := db.DB.First(&dbUser, user.UserID).Error; err != nil {
		log.Printf("Error fetching user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"User":          dbUser,
		"res_blog_name": getBlogName(),
		"ActiveTab":     "profile",
	}

	if err := renderTemplate(w, r, "public/profile.html", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func profileUpdate(w http.ResponseWriter, r *http.Request) {
	user := middleware.RequireUser(r)
	if user == nil {
		http.Redirect(w, r, "/access/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Redirect(w, r, "/profile", http.StatusFound)
		return
	}

	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")

	updates := map[string]interface{}{}
	if email != "" {
		updates["email"] = email
	}
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			middleware.GenerateFlashCookie(w, "Failed to update profile")
			http.Redirect(w, r, "/profile", http.StatusFound)
			return
		}
		updates["password"] = string(hash)
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&models.User{}).Where("id = ?", user.UserID).Updates(updates).Error; err != nil {
			log.Printf("Error updating user: %v", err)
			middleware.GenerateFlashCookie(w, "Failed to update profile")
		} else {
			middleware.GenerateFlashCookie(w, "Profile updated successfully")
		}
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

// getBlogName retrieves the blog name setting
func getBlogName() string {
	var setting models.SiteSetting
	if err := db.DB.Where("name = ?", "blog_name").First(&setting).Error; err != nil {
		return "ResCMS"
	}
	return setting.Value
}

// generateCSRFToken generates a basic CSRF token
func generateCSRFToken() string {
	return "csrf_token_placeholder"
}

// CSRF check placeholder
func checkCSRFToken(r *http.Request) bool {
	// In production, implement proper CSRF validation
	token := r.PostForm.Get("csrf_token")
	return strings.HasPrefix(token, "csrf_token")
}
