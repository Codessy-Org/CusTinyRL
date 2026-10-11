package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrGoogleDisabled reports that Google OAuth is not configured.
var ErrGoogleDisabled = errors.New("google login is not configured")

// ErrOAuthState reports an invalid or expired OAuth state parameter.
var ErrOAuthState = errors.New("invalid or expired oauth state")

// GoogleIdentity is the verified profile returned by Google's userinfo
// endpoint. Only verified emails are accepted.
type GoogleIdentity struct {
	Sub     string
	Email   string
	Name    string
	Picture string
}

// GoogleProvider implements the OAuth 2.0 authorization-code flow against
// Google (the only supported OAuth provider). The CSRF state is an
// HMAC-signed nonce+timestamp — stateless, single-instance safe, valid 10
// minutes.
type GoogleProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	frontendURL  string
	authURL      string // overridable for tests
	tokenURL     string
	userinfoURL  string
	httpClient   *http.Client
	now          func() time.Time
}

func NewGoogleProvider(clientID, clientSecret, redirectURL, frontendURL string) *GoogleProvider {
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		frontendURL:  frontendURL,
		authURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		tokenURL:     "https://oauth2.googleapis.com/token",
		userinfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		now:          time.Now,
	}
}

func (g *GoogleProvider) Enabled() bool {
	return g != nil && g.clientID != "" && g.clientSecret != ""
}

func (g *GoogleProvider) stateKey() []byte {
	// Derive the HMAC key from the client secret — no extra config surface.
	sum := sha256.Sum256([]byte("oauth-state:" + g.clientSecret))
	return sum[:]
}

func (g *GoogleProvider) signState(nonce string, ts int64) string {
	mac := hmac.New(sha256.New, g.stateKey())
	fmt.Fprintf(mac, "%s.%d", nonce, ts)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// makeState returns "<nonce>.<ts>.<sig>".
func (g *GoogleProvider) makeState() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(b[:])
	ts := g.now().Unix()
	return nonce + "." + strconv.FormatInt(ts, 10) + "." + g.signState(nonce, ts), nil
}

func (g *GoogleProvider) verifyState(state string) error {
	nonce, tsRaw, sig, ok := strings.Cut(state, ".")
	if !ok {
		return ErrOAuthState
	}
	ts, err := strconv.ParseInt(tsRaw, 10, 64)
	if err != nil {
		return ErrOAuthState
	}
	if g.now().Sub(time.Unix(ts, 0)) > 10*time.Minute || ts > g.now().Add(time.Minute).Unix() {
		return ErrOAuthState
	}
	want := g.signState(nonce, ts)
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return ErrOAuthState
	}
	return nil
}

// AuthURL returns the Google consent-screen URL for the given state.
func (g *GoogleProvider) AuthURL(state string) string {
	v := url.Values{
		"client_id":     {g.clientID},
		"redirect_uri":  {g.redirectURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"prompt":        {"select_account"},
	}
	return g.authURL + "?" + v.Encode()
}

// Exchange validates state, trades the authorization code for tokens, and
// returns the verified Google identity.
func (g *GoogleProvider) Exchange(ctx context.Context, code, state string) (GoogleIdentity, error) {
	if err := g.verifyState(state); err != nil {
		return GoogleIdentity{}, err
	}
	form := url.Values{
		"code":          {code},
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"redirect_uri":  {g.redirectURL},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return GoogleIdentity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return GoogleIdentity{}, fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return GoogleIdentity{}, errors.New("token exchange: malformed response")
	}

	ureq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userinfoURL, nil)
	if err != nil {
		return GoogleIdentity{}, err
	}
	ureq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	uresp, err := g.httpClient.Do(ureq)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("userinfo: %w", err)
	}
	defer uresp.Body.Close()
	ubody, _ := io.ReadAll(io.LimitReader(uresp.Body, 1<<20))
	if uresp.StatusCode != http.StatusOK {
		return GoogleIdentity{}, fmt.Errorf("userinfo failed: status %d", uresp.StatusCode)
	}
	var ui struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(ubody, &ui); err != nil {
		return GoogleIdentity{}, errors.New("userinfo: malformed response")
	}
	if ui.Sub == "" || ui.Email == "" || !ui.EmailVerified {
		return GoogleIdentity{}, errors.New("google account email is not verified")
	}
	return GoogleIdentity{Sub: ui.Sub, Email: normalizeEmail(ui.Email), Name: ui.Name, Picture: ui.Picture}, nil
}

// LoginWithGoogle creates/links the account and issues a session token.
func (s *Service) LoginWithGoogle(ctx context.Context, id GoogleIdentity) (*Token, error) {
	if !s.google.Enabled() {
		return nil, ErrGoogleDisabled
	}
	if err := validateEmail(id.Email); err != nil {
		return nil, err
	}
	u, err := s.repo.UpsertGoogleUser(ctx, id.Sub, id.Email, id.Name, id.Picture)
	if err != nil {
		return nil, err
	}
	return s.issueToken(ctx, u)
}
