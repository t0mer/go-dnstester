package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomerklein/dnstester/internal/auth"
	"github.com/tomerklein/dnstester/internal/config"
	"github.com/tomerklein/dnstester/internal/handler"
	"github.com/tomerklein/dnstester/internal/model"
)

// newTestEnv creates a temp config dir and returns cfgSvc, authSvc, authHandler.
func newTestEnv(t *testing.T) (*config.Service, *auth.Service, *handler.AuthHandler) {
	t.Helper()
	dir := t.TempDir()
	cfgSvc := config.NewService(dir)
	authSvc := auth.NewService()
	h := handler.NewAuthHandler(cfgSvc, authSvc)
	return cfgSvc, authSvc, h
}

// enableAuth saves a config with auth enabled, a known user, and a hashed password.
func enableAuth(t *testing.T, cfgSvc *config.Service, username, password string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	cfg := &model.Config{
		Auth: model.AuthConfig{
			Enabled:      true,
			Username:     username,
			PasswordHash: hash,
		},
	}
	if err := cfgSvc.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

// --- Status ---

func TestStatus_AuthDisabled(t *testing.T) {
	_, _, h := newTestEnv(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["auth_enabled"] != false {
		t.Error("expected auth_enabled=false")
	}
	if body["authenticated"] != true {
		t.Error("expected authenticated=true when auth is disabled")
	}
}

func TestStatus_AuthEnabled_Unauthenticated(t *testing.T) {
	cfgSvc, _, h := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	h.Status(rec, req)

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["auth_enabled"] != true {
		t.Error("expected auth_enabled=true")
	}
	if body["authenticated"] != false {
		t.Error("expected authenticated=false without a session")
	}
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	cfgSvc, _, h := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	body := `{"username":"alice","password":"password123"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie in response")
	}
	if sessionCookie.Value == "" {
		t.Error("session cookie value should not be empty")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	cfgSvc, _, h := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	body := `{"username":"alice","password":"wrongpassword"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestLogin_AuthDisabled(t *testing.T) {
	_, _, h := newTestEnv(t)

	body := `{"username":"alice","password":"password123"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

// --- Logout ---

func TestLogout_ClearsCookie(t *testing.T) {
	cfgSvc, authSvc, h := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	// Create a session to log out of.
	sessID, _ := authSvc.NewSession()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessID})
	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rec.Code)
	}
	// Session should be gone.
	if authSvc.ValidSession(sessID) {
		t.Error("session should be invalid after logout")
	}
}

// --- Guard middleware ---

func TestGuard_AuthDisabled_AllowsAll(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)
	guard := auth.Guard(cfgSvc, authSvc)

	called := false
	handler := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	handler(rec, req)

	if !called {
		t.Error("expected handler to be called when auth is disabled")
	}
}

func TestGuard_TokenOnlyMode_NoBearerRejected(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)

	// Token auth enabled, login NOT required.
	pt, hash, _ := auth.NewToken()
	_ = pt
	cfg := &model.Config{
		Auth: model.AuthConfig{
			Enabled:         false,
			APITokenEnabled: true,
			APITokenHash:    hash,
		},
	}
	cfgSvc.Save(cfg)

	guard := auth.Guard(cfgSvc, authSvc)
	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	// Request with NO auth → must be rejected.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	h(rec, req)

	if called {
		t.Error("expected handler NOT to be called without a token in token-only mode")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestGuard_TokenOnlyMode_ValidBearerAllowed(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)

	pt, hash, _ := auth.NewToken()
	cfgSvc.Save(&model.Config{
		Auth: model.AuthConfig{
			Enabled:         false,
			APITokenEnabled: true,
			APITokenHash:    hash,
		},
	})

	guard := auth.Guard(cfgSvc, authSvc)
	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	req.Header.Set("Authorization", "Bearer "+pt)
	h(rec, req)

	if !called {
		t.Error("expected handler to be called with a valid Bearer token in token-only mode")
	}
}

func TestGuard_AuthEnabled_NoCredentials_Rejects(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")
	guard := auth.Guard(cfgSvc, authSvc)

	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	h(rec, req)

	if called {
		t.Error("expected handler NOT to be called without auth")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestGuard_ValidSession_Allows(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")
	guard := auth.Guard(cfgSvc, authSvc)

	sessID, _ := authSvc.NewSession()

	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessID})
	h(rec, req)

	if !called {
		t.Error("expected handler to be called with valid session")
	}
}

func TestGuard_ValidBearerToken_Allows(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	// Set up a token in the config.
	pt, hash, _ := auth.NewToken()
	cfg, _ := cfgSvc.Load()
	cfg.Auth.APITokenEnabled = true
	cfg.Auth.APITokenHash = hash
	cfgSvc.Save(cfg)

	guard := auth.Guard(cfgSvc, authSvc)

	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	req.Header.Set("Authorization", "Bearer "+pt)
	h(rec, req)

	if !called {
		t.Error("expected handler to be called with valid bearer token")
	}
}

func TestGuard_InvalidBearerToken_Rejects(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	_, hash, _ := auth.NewToken()
	cfg, _ := cfgSvc.Load()
	cfg.Auth.APITokenEnabled = true
	cfg.Auth.APITokenHash = hash
	cfgSvc.Save(cfg)

	guard := auth.Guard(cfgSvc, authSvc)

	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	h(rec, req)

	if called {
		t.Error("expected handler NOT to be called with invalid token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestGuard_TokenAuthDisabled_BearerRejected(t *testing.T) {
	cfgSvc, authSvc, _ := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	// Token auth disabled even though hash exists.
	pt, hash, _ := auth.NewToken()
	cfg, _ := cfgSvc.Load()
	cfg.Auth.APITokenEnabled = false
	cfg.Auth.APITokenHash = hash
	cfgSvc.Save(cfg)

	guard := auth.Guard(cfgSvc, authSvc)

	called := false
	h := guard(func(w http.ResponseWriter, r *http.Request) { called = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test/run", nil)
	req.Header.Set("Authorization", "Bearer "+pt)
	h(rec, req)

	if called {
		t.Error("expected handler NOT to be called when API token auth is disabled")
	}
}

// --- Token generation and revocation ---

func TestGenerateAndRevokeToken(t *testing.T) {
	cfgSvc, authSvc, h := newTestEnv(t)
	enableAuth(t, cfgSvc, "alice", "password123")

	sessID, _ := authSvc.NewSession()
	addSession := func(req *http.Request) *http.Request {
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessID})
		return req
	}

	// Generate token.
	rec := httptest.NewRecorder()
	req := addSession(httptest.NewRequest(http.MethodPost, "/api/auth/token", nil))
	h.GenerateToken(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GenerateToken want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var tokenResp map[string]string
	json.NewDecoder(rec.Body).Decode(&tokenResp)
	pt := tokenResp["token"]
	if pt == "" {
		t.Fatal("expected non-empty token in response")
	}

	// Verify token is stored and works.
	cfg, _ := cfgSvc.Load()
	if cfg.Auth.APITokenHash == "" {
		t.Error("expected token hash to be persisted")
	}
	if !auth.CheckToken(pt, cfg.Auth.APITokenHash) {
		t.Error("stored hash does not match the returned plaintext token")
	}

	// Revoke token.
	rec2 := httptest.NewRecorder()
	req2 := addSession(httptest.NewRequest(http.MethodDelete, "/api/auth/token", nil))
	h.RevokeToken(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("RevokeToken want 204, got %d", rec2.Code)
	}

	cfg2, _ := cfgSvc.Load()
	if cfg2.Auth.APITokenHash != "" {
		t.Error("expected token hash to be cleared after revocation")
	}
	if cfg2.Auth.APITokenEnabled {
		t.Error("expected APITokenEnabled to be false after revocation (prevents UI lockout)")
	}
}

// --- UpdateSettings ---

func TestUpdateSettings_EnableAuth(t *testing.T) {
	cfgSvc, authSvc, h := newTestEnv(t)

	// Auth is not yet enabled — requireSession should pass.
	rec := httptest.NewRecorder()
	body := `{"enabled":true,"username":"bob","password":"supersecret","api_token_enabled":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/auth/settings", strings.NewReader(body))
	h.UpdateSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cfg, _ := cfgSvc.Load()
	if !cfg.Auth.Enabled {
		t.Error("expected auth to be enabled")
	}
	if cfg.Auth.Username != "bob" {
		t.Errorf("expected username=bob, got %q", cfg.Auth.Username)
	}
	if cfg.Auth.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}
	if !auth.CheckPassword("supersecret", cfg.Auth.PasswordHash) {
		t.Error("password hash does not match")
	}

	// Password change should clear sessions.
	sessID, _ := authSvc.NewSession()
	if !authSvc.ValidSession(sessID) {
		t.Error("session should be valid before password change")
	}

	// Change password.
	rec2 := httptest.NewRecorder()
	body2 := `{"enabled":true,"username":"bob","password":"newpassword1","api_token_enabled":false}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/auth/settings", strings.NewReader(body2))
	req2.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessID})
	h.UpdateSettings(rec2, req2)

	if authSvc.ValidSession(sessID) {
		t.Error("sessions should be invalidated after password change")
	}
}

func TestUpdateSettings_ValidationErrors(t *testing.T) {
	_, _, h := newTestEnv(t)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"empty username", `{"enabled":true,"username":"","password":"long-enough","api_token_enabled":false}`, http.StatusBadRequest},
		{"short password", `{"enabled":true,"username":"alice","password":"short","api_token_enabled":false}`, http.StatusBadRequest},
		{"missing password on first enable", `{"enabled":true,"username":"alice","password":"","api_token_enabled":false}`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/auth/settings", strings.NewReader(tc.body))
			h.UpdateSettings(rec, req)
			if rec.Code != tc.want {
				t.Errorf("want %d, got %d: %s", tc.want, rec.Code, rec.Body.String())
			}
		})
	}
}

// --- Config handler does not expose secret fields ---

func TestConfigHandlerGet_MasksSecrets(t *testing.T) {
	dir := t.TempDir()
	cfgSvc := config.NewService(dir)

	// Save a config with secrets.
	cfgSvc.Save(&model.Config{
		Auth: model.AuthConfig{
			Enabled:      true,
			PasswordHash: "should-not-appear",
			APITokenHash: "should-not-appear-either",
		},
	})

	h := handler.NewConfigHandler(cfgSvc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	h.Get(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "should-not-appear") {
		t.Error("secret fields must not be returned by GET /api/config")
	}
}

func TestConfigHandlerUpdate_PreservesAuthSecrets(t *testing.T) {
	dir := t.TempDir()

	// Write a config file directly so the hash field exists on disk.
	raw := `{"auth":{"enabled":true,"password_hash":"keep-this","api_token_hash":"keep-this-too"}}`
	os.WriteFile(filepath.Join(dir, "dnstester.json"), []byte(raw), 0644)

	cfgSvc := config.NewService(dir)
	h := handler.NewConfigHandler(cfgSvc)

	// Update config — the incoming payload has no auth field (or empty auth).
	rec := httptest.NewRecorder()
	body := `{"servers":[],"fqdns":[],"schedules":[],"auto_update":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(body))
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	// Verify secrets were preserved on disk.
	cfg, _ := cfgSvc.Load()
	if cfg.Auth.PasswordHash != "keep-this" {
		t.Errorf("password_hash should be preserved, got %q", cfg.Auth.PasswordHash)
	}
	if cfg.Auth.APITokenHash != "keep-this-too" {
		t.Errorf("api_token_hash should be preserved, got %q", cfg.Auth.APITokenHash)
	}
}
