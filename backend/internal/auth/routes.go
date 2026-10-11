package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/config"
	"github.com/gauravkrrr/url-shortner/backend/internal/middleware"
	"github.com/gauravkrrr/url-shortner/backend/internal/ratelimit"
)

// Routes mounts /api/v1/auth and /api/v1/me.
func Routes(h *Handler, requireAuth func(http.Handler) http.Handler,
	limiter *ratelimit.Limiter, cfg *config.Config) http.Handler {

	r := chi.NewRouter()
	policy := cfg.RateLimits["auth"]
	byIP := func(req *http.Request) string {
		return "ip:" + middleware.GetClientIP(req.Context())
	}

	r.With(middleware.RateLimit(limiter, policy, byIP)).Post("/register", h.Register)
	r.With(middleware.RateLimit(limiter, policy, byIP)).Post("/login", h.Login)
	r.With(requireAuth).Post("/logout", h.Logout)

	r.Get("/providers", h.Providers)
	r.With(middleware.RateLimit(limiter, policy, byIP)).Get("/google", h.GoogleStart)
	r.Get("/google/callback", h.GoogleCallback)

	return r
}
