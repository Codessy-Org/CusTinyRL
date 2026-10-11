package apikey

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

type createRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at"`
}

type keyResponse struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	KeyPrefix  string   `json:"key_prefix"`
	Scopes     []string `json:"scopes"`
	CreatedAt  string   `json:"created_at"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	RevokedAt  *string  `json:"revoked_at,omitempty"`
}

func toKeyResponse(k Key) keyResponse {
	resp := keyResponse{
		ID:        k.ID,
		Name:      k.Name,
		KeyPrefix: k.KeyPrefix,
		Scopes:    k.Scopes,
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
	}
	if k.ExpiresAt != nil {
		v := k.ExpiresAt.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &v
	}
	if k.LastUsedAt != nil {
		v := k.LastUsedAt.UTC().Format(time.RFC3339)
		resp.LastUsedAt = &v
	}
	if k.RevokedAt != nil {
		v := k.RevokedAt.UTC().Format(time.RFC3339)
		resp.RevokedAt = &v
	}
	return resp
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := identity.UserID(r.Context())
	if userID == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req createRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil || !t.After(time.Now()) {
			httpx.Error(w, http.StatusUnprocessableEntity, "validation_error", "expires_at must be a future RFC3339 timestamp")
			return
		}
		expiresAt = &t
	}
	created, err := h.svc.Create(r.Context(), *userID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"key":  created.Raw, // shown exactly once
		"data": toKeyResponse(created.Key),
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := identity.UserID(r.Context())
	if userID == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	keys, err := h.svc.List(r.Context(), *userID)
	if err != nil {
		h.log.Error("list api keys", "err", err.Error())
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	data := make([]keyResponse, 0, len(keys))
	for _, k := range keys {
		data = append(data, toKeyResponse(k))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	userID := identity.UserID(r.Context())
	if userID == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid key id")
		return
	}
	if err := h.svc.Revoke(r.Context(), *userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "not_found", "api key not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

// Routes mounts key management. These endpoints require a session token
// (RequireSession) — API keys cannot mint or revoke other keys.
func Routes(h *Handler, requireSession func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.With(requireSession).Get("/", h.List)
	r.With(requireSession).Post("/", h.Create)
	r.With(requireSession).Delete("/{id}", h.Revoke)
	return r
}
