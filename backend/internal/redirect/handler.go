package redirect

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/metrics"
	"github.com/gauravkrrr/url-shortner/backend/internal/url"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Go handles GET /{slug}.
func (h *Handler) Go(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	metrics.Registry.RedirectTotal.Inc()

	slug := chi.URLParam(r, "slug")
	canonical, ok := url.ValidateRedirectSlug(slug)
	if !ok {
		metrics.Registry.RedirectErrors.Inc()
		httpx.Error(w, http.StatusNotFound, "not_found", "short link not found")
		return
	}

	entry, outcome, err := h.svc.Resolve(r.Context(), canonical)
	if err != nil {
		metrics.Registry.RedirectErrors.Inc()
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to resolve short link")
		return
	}

	switch outcome {
	case NotFound:
		metrics.Registry.RedirectErrors.Inc()
		httpx.Error(w, http.StatusNotFound, "not_found", "short link not found")
		return
	case Inactive:
		httpx.Error(w, http.StatusGone, "gone", "this short link has been disabled")
		return
	case Expired:
		httpx.Error(w, http.StatusGone, "gone", "this short link has expired")
		return
	}

	h.svc.RecordClick(entry, r.Referer(), r.Header.Get("User-Agent"))
	metrics.Registry.RedirectLatency.Observe(time.Since(start))
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, entry.OriginalURL, http.StatusFound)
}
