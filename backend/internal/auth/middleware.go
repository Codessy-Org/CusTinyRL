package auth

import (
	"net/http"
	"strings"

	"github.com/gauravkrrr/url-shortner/backend/internal/apikey"
	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
)

// Bearer extracts the raw bearer token from the Authorization header.
func Bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// SetAPIKeyService enables API-key authentication alongside session tokens.
// Without it, only session tokens are accepted.
func (s *Service) SetAPIKeyService(keys *apikey.Service) { s.keys = keys }

// OptionalAuth populates the user id when a valid credential is present
// (session token or API key with any scopes). Invalid session tokens fall
// through to anonymous; API keys fail loudly — a revoked or expired machine
// credential must never silently degrade to anonymous access.
func (s *Service) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := Bearer(r)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		if apikey.LooksLikeKey(raw) {
			if s.keys == nil {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			p, err := s.keys.Authenticate(r.Context(), raw)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			ctx := identity.WithUserID(r.Context(), p.UserID)
			ctx = identity.WithScopes(ctx, p.Scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if u, err := s.Authenticate(r.Context(), raw); err == nil {
			next.ServeHTTP(w, r.WithContext(identity.WithUserID(r.Context(), u.ID)))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSession rejects anything that is not a valid session token.
// Used for credential-management endpoints (API keys) so keys cannot mint
// further credentials.
func (s *Service) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := Bearer(r)
		if raw == "" || apikey.LooksLikeKey(raw) {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "session authentication required")
			return
		}
		u, err := s.Authenticate(r.Context(), raw)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(identity.WithUserID(r.Context(), u.ID)))
	})
}

// RequireAuth rejects requests without a valid session token or API key.
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := Bearer(r)
		if raw == "" {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if apikey.LooksLikeKey(raw) {
			if s.keys == nil {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			p, err := s.keys.Authenticate(r.Context(), raw)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			ctx := identity.WithUserID(r.Context(), p.UserID)
			ctx = identity.WithScopes(ctx, p.Scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		u, err := s.Authenticate(r.Context(), raw)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(identity.WithUserID(r.Context(), u.ID)))
	})
}
