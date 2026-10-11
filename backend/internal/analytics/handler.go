package analytics

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
	"github.com/gauravkrrr/url-shortner/backend/internal/url"
)

type Handler struct {
	querier *Querier
	urls    *url.Service
}

func NewHandler(querier *Querier, urls *url.Service) *Handler {
	return &Handler{querier: querier, urls: urls}
}

func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid url id")
		return
	}
	userID := identity.UserID(r.Context())
	if _, err := h.urls.Get(r.Context(), id, userID); err != nil {
		httpx.Error(w, http.StatusNotFound, "not_found", "url not found")
		return
	}
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil {
			days = n
		}
	}
	summary, err := h.querier.Summary(r.Context(), id, days)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load analytics")
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}
