package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler { return &Handler{svc: svc, log: log} }

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	User      User   `json:"user"`
}

func toTokenResponse(t *Token) tokenResponse {
	return tokenResponse{
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		User:      t.User,
	}
}

func writeAuthErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, "email_taken", "an account with this email already exists")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	default:
		var msg string
		if err.Error() != "" {
			msg = err.Error()
		} else {
			msg = "authentication failed"
		}
		// Validation-style errors from Register.
		if msg == "password must be at least 8 characters" || msg == "invalid email address" {
			httpx.Error(w, http.StatusUnprocessableEntity, "validation_error", msg)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := httpx.Decode(r, &c); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	t, err := h.svc.Register(r.Context(), c.Email, c.Password)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toTokenResponse(t))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := httpx.Decode(r, &c); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	t, err := h.svc.Login(r.Context(), c.Email, c.Password)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toTokenResponse(t))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), Bearer(r)); err != nil {
		h.log.Warn("logout failed", "err", err.Error())
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	uid := identity.UserID(r.Context())
	if uid == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	u, err := h.svc.Authenticate(r.Context(), Bearer(r))
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"user": u})
}

// Providers reports which OAuth providers are configured, so the frontend
// can hide disabled sign-in options.
func (h *Handler) Providers(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"google": h.svc.google.Enabled()})
}

// GoogleStart redirects to Google's consent screen.
func (h *Handler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	g := h.svc.google
	if !g.Enabled() {
		httpx.Error(w, http.StatusNotImplemented, "google_auth_disabled",
			"set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET to enable Google login")
		return
	}
	state, err := g.makeState()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	http.Redirect(w, r, g.AuthURL(state), http.StatusFound)
}

// GoogleCallback finishes the flow and lands on the dashboard with the
// session token in the URL fragment (read once by the SPA, never sent to
// the server in requests).
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	g := h.svc.google
	fail := func(msg string) {
		http.Redirect(w, r, g.frontendURL+"/dashboard#oauth_error="+url.QueryEscape(msg), http.StatusFound)
	}
	if !g.Enabled() {
		fail("google login is not configured")
		return
	}
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		fail("google denied access")
		return
	}
	code := q.Get("code")
	if code == "" {
		fail("missing authorization code")
		return
	}
	id, err := g.Exchange(r.Context(), code, q.Get("state"))
	if err != nil {
		h.log.Warn("google oauth exchange failed", "err", err.Error())
		fail("could not verify google login")
		return
	}
	t, err := h.svc.LoginWithGoogle(r.Context(), id)
	if err != nil {
		h.log.Warn("google login failed", "err", err.Error())
		fail("could not sign in")
		return
	}
	http.Redirect(w, r, g.frontendURL+"/dashboard#token="+url.QueryEscape(t.Token), http.StatusFound)
}
