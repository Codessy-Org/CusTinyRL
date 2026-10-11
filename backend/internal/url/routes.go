package url

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/config"
	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
	"github.com/gauravkrrr/url-shortner/backend/internal/middleware"
	"github.com/gauravkrrr/url-shortner/backend/internal/ratelimit"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func itoaSec(d time.Duration) string {
	s := int(d.Seconds())
	if s < 1 {
		s = 1
	}
	return strconv.Itoa(s)
}

// Routes mounts the URL-management API. Create allows anonymous callers only
// when cfg.AllowAnonymous; management endpoints always require authentication.
// analyticsHandler, when non-nil, is mounted at GET /{id}/analytics.
func Routes(h *Handler, requireAuth func(http.Handler) http.Handler, optionalAuth func(http.Handler) http.Handler,
	limiter *ratelimit.Limiter, cfg *config.Config, analyticsHandler http.Handler) http.Handler {

	r := chi.NewRouter()

	createMW := requireAuth
	if cfg.AllowAnonymous {
		createMW = optionalAuth
	}

	r.With(
		middleware.RateLimit(limiter, cfg.RateLimits["url_create"], func(r *http.Request) string {
			if uid := identity.UserID(r.Context()); uid != nil {
				return "user:" + itoa(*uid)
			}
			return "ip:" + middleware.GetClientIP(r.Context())
		}),
		func(next http.Handler) http.Handler {
			// Anonymous creation gets a stricter IP-based policy on top.
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if identity.UserID(r.Context()) == nil {
					res, err := limiter.Allow(r.Context(), cfg.RateLimits["anon_create"], "ip:"+middleware.GetClientIP(r.Context()))
					if err == nil && !res.Allowed {
						w.Header().Set("Retry-After", itoaSec(res.RetryAfter))
						httpx.Error(w, http.StatusTooManyRequests, "rate_limited", "too many requests, slow down")
						return
					}
				}
				next.ServeHTTP(w, r)
			})
		},
		middleware.RequireScope("create"),
		createMW,
	).Post("/", h.Create)

	r.With(requireAuth, middleware.RequireScope("read")).Get("/", h.List)
	r.With(requireAuth, middleware.RequireScope("read")).Get("/{id}", h.Get)
	r.With(requireAuth, middleware.RequireScope("manage")).Patch("/{id}", h.Update)
	r.With(requireAuth, middleware.RequireScope("manage")).Delete("/{id}", h.Delete)

	if analyticsHandler != nil {
		r.With(requireAuth, middleware.RequireScope("read"), middleware.RateLimit(limiter, cfg.RateLimits["analytics"],
			func(req *http.Request) string {
				if uid := identity.UserID(req.Context()); uid != nil {
					return "user:" + itoa(*uid)
				}
				return "ip:" + middleware.GetClientIP(req.Context())
			},
		)).Get("/{id}/analytics", analyticsHandler.ServeHTTP)
	}

	return r
}
