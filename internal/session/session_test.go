package session

import (
	"testing"
	"time"
)

func TestSessionStore(t *testing.T) {
	secrets := []string{"test-secret-key-that-is-long-enough"}
	duration := 1 * time.Hour
	store := New(secrets, duration)

	// Test Create
	sess, err := store.Create(1, "testuser", false)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if sess.UserID != 1 || sess.Username != "testuser" {
		t.Errorf("session data mismatch")
	}

	// Test Get
	fetched, ok := store.Get(sess.ID)
	if !ok {
		t.Fatal("failed to get session")
	}
	if fetched.ID != sess.ID {
		t.Errorf("fetched session ID mismatch")
	}

	// Test Encode/Decode
	encoded, err := store.Encode(sess)
	if err != nil {
		t.Fatalf("failed to encode session: %v", err)
	}

	decoded, err := store.Decode(encoded)
	if err != nil {
		t.Fatalf("failed to decode session: %v", err)
	}

	if decoded.ID != sess.ID || decoded.UserID != sess.UserID {
		t.Errorf("decoded session mismatch")
	}

	// Test Destroy
	store.Destroy(sess.ID)
	_, ok = store.Get(sess.ID)
	if ok {
		t.Error("session still exists after destroy")
	}
}

func TestSessionExpiration(t *testing.T) {
	secrets := []string{"test-secret"}
	duration := 1 * time.Millisecond
	store := New(secrets, duration)

	sess, _ := store.Create(1, "testuser", false)
	
	// Wait for expiration
	time.Sleep(2 * time.Millisecond)

	_, ok := store.Get(sess.ID)
	if ok {
		t.Error("session should have expired")
	}

	// Test Cleanup
	store.Cleanup()
	store.mu.RLock()
	_, ok = store.sessions[sess.ID]
	store.mu.RUnlock()
	if ok {
		t.Error("expired session not removed during cleanup")
	}
}
