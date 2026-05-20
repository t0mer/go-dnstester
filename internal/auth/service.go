package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "dnst_session"
	SessionTTL        = 24 * time.Hour
	MinPasswordLength = 8
	bcryptCost        = 12
)

type sessionEntry struct {
	expiry time.Time
}

// Service manages sessions and provides auth primitives.
type Service struct {
	mu       sync.RWMutex
	sessions map[string]sessionEntry
}

func NewService() *Service {
	return &Service{sessions: make(map[string]sessionEntry)}
}

// HashPassword returns a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// CheckPassword reports whether password matches the bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// NewSession creates a new session and returns its ID.
func (s *Service) NewSession() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	id := hex.EncodeToString(b)
	s.mu.Lock()
	s.sessions[id] = sessionEntry{expiry: time.Now().Add(SessionTTL)}
	s.mu.Unlock()
	return id, nil
}

// ValidSession reports whether id is a known, non-expired session.
func (s *Service) ValidSession(id string) bool {
	s.mu.RLock()
	entry, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(entry.expiry) {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		return false
	}
	return true
}

// DeleteSession removes a single session.
func (s *Service) DeleteSession(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// ClearAllSessions invalidates every active session (e.g. after a password change).
func (s *Service) ClearAllSessions() {
	s.mu.Lock()
	s.sessions = make(map[string]sessionEntry)
	s.mu.Unlock()
}

// NewToken generates a random API token. It returns the plaintext (shown once to the
// user) and a SHA-256 hex hash (stored on disk).
func NewToken() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	plaintext = hex.EncodeToString(b)
	hash = HashToken(plaintext)
	return plaintext, hash, nil
}

// HashToken returns the SHA-256 hex hash of a token plaintext.
func HashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// CheckToken reports whether plaintext matches the stored hash.
func CheckToken(plaintext, hash string) bool {
	if plaintext == "" || hash == "" {
		return false
	}
	return HashToken(plaintext) == hash
}
