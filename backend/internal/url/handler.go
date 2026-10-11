package url

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
	"github.com/gauravkrrr/url-shortner/backend/internal/metrics"
)

type Handler struct {
	svc     *Service
	baseURL string
	log     *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, baseURL: svc.BaseURL(), log: log}
}

func toResponse(s *ShortURL, baseURL string) URLResponse {
	var expires *string
	if s.ExpiresAt != nil {
		v := s.ExpiresAt.UTC().Format(time.RFC3339)
		expires = &v
	}
	now := time.Now()
	isExpired := s.ExpiresAt != nil && !s.ExpiresAt.After(now)
	return URLResponse{
		ID:          s.ID,
		Slug:        s.Slug,
		ShortURL:    baseURL + "/" + s.Slug,
		OriginalURL: s.OriginalURL,
		Title:       s.Title,
		FaviconURL:  s.FaviconURL,
		IsActive:    s.IsActive,
		IsExpired:   isExpired,
		ClickCount:  s.ClickCount,
		ExpiresAt:   expires,
		CreatedAt:   s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var verr ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.Error(w, http.StatusUnprocessableEntity, "validation_error", verr.Msg)
	case errors.Is(err, ErrSlugTaken):
		metrics.Registry.URLsCreatedRejected.Inc()
		httpx.Error(w, http.StatusConflict, "slug_taken", "that slug is already taken")
	case errors.Is(err, ErrAuthRequired):
		httpx.Error(w, http.StatusForbidden, "auth_required", "sign in to choose a custom slug or prefix")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "url not found")
	default:
		h.log.Error("url handler error", "err", err.Error())
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func parseExpiry(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, ValidationError{Msg: "expires_at must be an RFC3339 timestamp"}
	}
	if !t.After(time.Now()) {
		return nil, ValidationError{Msg: "expires_at must be in the future"}
	}
	return &t, nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateURLRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	expires, err := parseExpiry(req.ExpiresAt)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	rec, err := h.svc.Create(r.Context(), CreateInput{
		UserID:      identity.UserID(r.Context()),
		OriginalURL: req.OriginalURL,
		CustomSlug:  req.CustomSlug,
		Prefix:      req.Prefix,
		Title:       req.Title,
		ExpiresAt:   expires,
	})
	if err != nil {
		h.writeErr(w, err)
		return
	}
	metrics.Registry.URLsCreated.Inc()
	httpx.JSON(w, http.StatusCreated, toResponse(rec, h.baseURL))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := identity.UserID(r.Context())
	if userID == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var cursor *int64
	if c := r.URL.Query().Get("cursor"); c != "" {
		if v, err := strconv.ParseInt(c, 10, 64); err == nil {
			cursor = &v
		}
	}
	items, next, err := h.svc.List(r.Context(), userID, cursor, limit, r.URL.Query().Get("q"))
	if err != nil {
		h.log.Error("list urls", "err", err.Error())
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := URLListResponse{Data: []URLResponse{}}
	for i := range items {
		resp.Data = append(resp.Data, toResponse(&items[i], h.baseURL))
	}
	if next != nil {
		v := strconv.FormatInt(*next, 10)
		resp.NextCursor = &v
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid url id")
		return
	}
	rec, err := h.svc.Get(r.Context(), id, identity.UserID(r.Context()))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(rec, h.baseURL))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid url id")
		return
	}
	var req UpdateURLRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	in := UpdateInput{
		UserID:      identity.UserID(r.Context()),
		OriginalURL: req.OriginalURL,
		IsActive:    req.IsActive,
		Title:       req.Title,
	}
	if req.ClearExpiry != nil && *req.ClearExpiry {
		in.ClearExpiry = true
	} else if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := parseExpiry(req.ExpiresAt)
		if err != nil {
			h.writeErr(w, err)
			return
		}
		in.ExpiresAt = t
	}
	if req.Title != nil && *req.Title == "" {
		in.ClearTitle = true
		in.Title = nil
	}
	rec, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toResponse(rec, h.baseURL))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid url id")
		return
	}
	if err := h.svc.Delete(r.Context(), id, identity.UserID(r.Context())); err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}
