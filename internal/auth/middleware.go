package auth

import (
	"net/http"
	"strings"

	"github.com/tomerklein/dnstester/internal/model"
)

// configLoader is satisfied by *config.Service; defined here to avoid an import cycle.
type configLoader interface {
	Load() (*model.Config, error)
}

// Guard returns a middleware that enforces authentication when any auth mode is active.
//
// Auth is required when:
//   - cfg.Auth.Enabled is true (login required for browser/UI access), OR
//   - cfg.Auth.APITokenEnabled is true AND a token hash exists (external API protection)
//
// Accepted credentials:
//   - A valid session cookie (created at login, covers browser/UI access)
//   - A valid Bearer token in the Authorization header (covers external API clients
//     and the browser UI when it sends the token from its local storage)
func Guard(cfgSvc configLoader, svc *Service) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cfg, err := cfgSvc.Load()
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			tokenActive := cfg.Auth.APITokenEnabled && cfg.Auth.APITokenHash != ""
			if !cfg.Auth.Enabled && !tokenActive {
				next(w, r)
				return
			}

			// Session cookie (browser/UI users who have logged in)
			if c, err := r.Cookie(SessionCookieName); err == nil && svc.ValidSession(c.Value) {
				next(w, r)
				return
			}

			// Bearer token (external API clients, and the UI when it holds the token)
			if tokenActive {
				if hdr := r.Header.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
					tok := strings.TrimPrefix(hdr, "Bearer ")
					if CheckToken(tok, cfg.Auth.APITokenHash) {
						next(w, r)
						return
					}
				}
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
	}
}
