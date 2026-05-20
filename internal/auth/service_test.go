package auth

import (
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword("correct-horse", hash) {
		t.Error("CheckPassword: expected true for correct password")
	}
	if CheckPassword("wrong", hash) {
		t.Error("CheckPassword: expected false for wrong password")
	}
}

func TestNewSession_ValidAndExpiry(t *testing.T) {
	svc := NewService()

	id, err := svc.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if id == "" {
		t.Fatal("NewSession returned empty id")
	}
	if !svc.ValidSession(id) {
		t.Error("ValidSession: expected true for new session")
	}

	// Unknown id should be invalid.
	if svc.ValidSession("nonexistent") {
		t.Error("ValidSession: expected false for unknown id")
	}
}

func TestDeleteSession(t *testing.T) {
	svc := NewService()
	id, _ := svc.NewSession()
	svc.DeleteSession(id)
	if svc.ValidSession(id) {
		t.Error("ValidSession: expected false after DeleteSession")
	}
}

func TestClearAllSessions(t *testing.T) {
	svc := NewService()
	id1, _ := svc.NewSession()
	id2, _ := svc.NewSession()
	svc.ClearAllSessions()
	if svc.ValidSession(id1) || svc.ValidSession(id2) {
		t.Error("ValidSession: expected false after ClearAllSessions")
	}
}

func TestExpiredSession(t *testing.T) {
	svc := NewService()
	id, _ := svc.NewSession()
	// Manually expire the session.
	svc.mu.Lock()
	svc.sessions[id] = sessionEntry{expiry: time.Now().Add(-time.Second)}
	svc.mu.Unlock()
	if svc.ValidSession(id) {
		t.Error("ValidSession: expected false for expired session")
	}
}

func TestToken(t *testing.T) {
	pt, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if pt == "" || hash == "" {
		t.Fatal("NewToken returned empty strings")
	}
	if !CheckToken(pt, hash) {
		t.Error("CheckToken: expected true for correct plaintext")
	}
	if CheckToken("wrong", hash) {
		t.Error("CheckToken: expected false for wrong plaintext")
	}
	if CheckToken("", hash) {
		t.Error("CheckToken: expected false for empty plaintext")
	}
	if CheckToken(pt, "") {
		t.Error("CheckToken: expected false for empty hash")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	// Same input must always produce the same hash.
	if HashToken("abc") != HashToken("abc") {
		t.Error("HashToken is not deterministic")
	}
	if HashToken("abc") == HashToken("xyz") {
		t.Error("HashToken collision for different inputs")
	}
}
