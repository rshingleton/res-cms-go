package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    uint
	Username  string
	IsAdmin   bool
	Data      map[string]interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store manages sessions in memory
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	Secrets  []string
	duration time.Duration
}

// New creates a new session store
func New(secrets []string, duration time.Duration) *Store {
	return &Store{
		sessions: make(map[string]*Session),
		Secrets:  secrets,
		duration: duration,
	}
}

// Get retrieves a session by ID
func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// Create creates a new session
func (s *Store) Create(userID uint, username string, isAdmin bool) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	session := &Session{
		ID:        id,
		UserID:    userID,
		Username:  username,
		IsAdmin:   isAdmin,
		Data:      make(map[string]interface{}),
		CreatedAt: now,
		ExpiresAt: now.Add(s.duration),
	}

	s.sessions[id] = session

	return session, nil
}

// Destroy removes a session
func (s *Store) Destroy(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
}

// Renew updates the session expiration
func (s *Store) Renew(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}

	session.ExpiresAt = time.Now().Add(s.duration)
	return nil
}

// Cleanup removes expired sessions
func (s *Store) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

// StartCleanup starts periodic cleanup of expired sessions
func (s *Store) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.Cleanup()
		}
	}()
}

// Encode encodes session data for storage in cookie
func (s *Store) Encode(session *Session) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	key := s.getSigningKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decode decodes session data from cookie
func (s *Store) Decode(encoded string) (*Session, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	key := s.getSigningKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(plaintext, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Store) getSigningKey() []byte {
	hash := sha256.Sum256([]byte(s.Secrets[0]))
	return hash[:]
}

// DefaultStore is the global session store
var DefaultStore *Store

// Init initializes the session store
func Init(secrets []string) {
	DefaultStore = New(secrets, 24*time.Hour)
	DefaultStore.StartCleanup(10 * time.Minute)
}

// Get returns the default session store
func Get() *Store {
	return DefaultStore
}
