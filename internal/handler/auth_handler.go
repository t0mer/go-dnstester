package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tomerklein/dnstester/internal/auth"
	"github.com/tomerklein/dnstester/internal/config"
)

// AuthHandler handles all /api/auth/* endpoints.
type AuthHandler struct {
	cfgSvc  *config.Service
	authSvc *auth.Service
}

func NewAuthHandler(cfgSvc *config.Service, authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{cfgSvc: cfgSvc, authSvc: authSvc}
}

// Status returns current auth state. Always public — the UI polls this to decide
// whether to show the login page.
// GET /api/auth/status
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tokenActive := cfg.Auth.APITokenEnabled && cfg.Auth.APITokenHash != ""
	authRequired := cfg.Auth.Enabled || tokenActive

	// No auth configured at all — always authenticated.
	authenticated := !authRequired
	if authRequired {
		// Session cookie (browser/UI with login)
		if c, err := r.Cookie(auth.SessionCookieName); err == nil {
			authenticated = h.authSvc.ValidSession(c.Value)
		}
		// Bearer token (external clients or UI with stored token)
		if !authenticated && tokenActive {
			if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
				tok := strings.TrimPrefix(hdr, "Bearer ")
				authenticated = auth.CheckToken(tok, cfg.Auth.APITokenHash)
			}
		}
	}

	writeJSON(w, map[string]any{
		"auth_enabled":      cfg.Auth.Enabled,
		"api_token_enabled": cfg.Auth.APITokenEnabled,
		"has_credentials":   cfg.Auth.PasswordHash != "",
		"has_token":         cfg.Auth.APITokenHash != "",
		"authenticated":     authenticated,
		"username":          cfg.Auth.Username,
	})
}

// Login validates credentials and issues a session cookie.
// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !cfg.Auth.Enabled {
		http.Error(w, "authentication is not enabled", http.StatusBadRequest)
		return
	}
	if body.Username != cfg.Auth.Username || !auth.CheckPassword(body.Password, cfg.Auth.PasswordHash) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	id, err := h.authSvc.NewSession()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.SessionTTL.Seconds()),
	})
	writeJSON(w, map[string]bool{"ok": true})
}

// Logout clears the session cookie and invalidates the session.
// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil {
		h.authSvc.DeleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

// UpdateSettings enables/disables auth and updates credentials. Requires a valid session.
// PUT /api/auth/settings
func (h *AuthHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}

	var body struct {
		Enabled         bool   `json:"enabled"`
		Username        string `json:"username"`
		Password        string `json:"password"` // empty = keep existing hash
		APITokenEnabled bool   `json:"api_token_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if body.Enabled && strings.TrimSpace(body.Username) == "" {
		http.Error(w, "username cannot be empty", http.StatusBadRequest)
		return
	}
	if body.Enabled && body.Password != "" && len(body.Password) < auth.MinPasswordLength {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Enabling for the first time requires a password.
	if body.Enabled && cfg.Auth.PasswordHash == "" && body.Password == "" {
		http.Error(w, "a password is required to enable authentication", http.StatusBadRequest)
		return
	}

	passwordChanged := false
	cfg.Auth.Enabled = body.Enabled
	cfg.Auth.APITokenEnabled = body.APITokenEnabled
	if body.Enabled {
		cfg.Auth.Username = strings.TrimSpace(body.Username)
		if body.Password != "" {
			hash, err := auth.HashPassword(body.Password)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			cfg.Auth.PasswordHash = hash
			passwordChanged = true
		}
	}

	if err := h.cfgSvc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if passwordChanged {
		h.authSvc.ClearAllSessions()
	}

	writeJSON(w, map[string]any{
		"enabled":           cfg.Auth.Enabled,
		"username":          cfg.Auth.Username,
		"api_token_enabled": cfg.Auth.APITokenEnabled,
		"has_token":         cfg.Auth.APITokenHash != "",
	})
}

// GenerateToken creates a new API token and returns the plaintext (shown once).
// POST /api/auth/token
func (h *AuthHandler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}

	plaintext, hash, err := auth.NewToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	cfg.Auth.APITokenHash = hash
	if err := h.cfgSvc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"token": plaintext})
}

// RevokeToken removes the stored token hash and disables API token auth.
// Also disables APITokenEnabled so that revoking without a login session does not
// lock the user out of the UI (the guard only enforces auth when at least one
// mode is active).
// DELETE /api/auth/token
func (h *AuthHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}

	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	cfg.Auth.APITokenHash = ""
	cfg.Auth.APITokenEnabled = false
	if err := h.cfgSvc.Save(cfg); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireSession returns true when the request carries a valid browser session
// (or when auth is disabled). It writes 401 and returns false otherwise.
func (h *AuthHandler) requireSession(w http.ResponseWriter, r *http.Request) bool {
	cfg, err := h.cfgSvc.Load()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return false
	}
	if !cfg.Auth.Enabled {
		return true
	}
	c, err := r.Cookie(auth.SessionCookieName)
	if err != nil || !h.authSvc.ValidSession(c.Value) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}
