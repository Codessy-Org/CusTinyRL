// Package apikey provides scoped, revocable API keys for non-interactive
// clients (CLI, scripts, CI). Keys are shown once at creation; only a
// SHA-256 digest is stored. Session tokens keep the `ck_` prefix free.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const Prefix = "ck_"

var (
	ErrNotFound = errors.New("api key not found")
	ErrInvalid  = errors.New("invalid api key")
	ErrNoScope  = errors.New("scope required")
	namePattern = regexp.MustCompile(`^[\w][\w .\-]{0,62}$`)
	validScopes = []string{"create", "read", "manage"}
)

type Key struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// Created wraps a key with the raw secret, present only in the create response.
type Created struct {
	Key
	Raw string `json:"key"`
}

// Principal is the authenticated caller resolved from a raw key.
type Principal struct {
	UserID int64
	Scopes []string
	KeyID  int64
}

func hashKey(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func newRawKey() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return Prefix + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// ValidateScopes normalizes and checks a scope list. "admin" expands to all.
func ValidateScopes(scopes []string) ([]string, error) {
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if s == "admin" {
			return slices.Clone(validScopes), nil
		}
		if !slices.Contains(validScopes, s) {
			return nil, fmt.Errorf("%w: unknown scope %q (use create, read, manage, or admin)", ErrNoScope, s)
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	slices.Sort(out)
	if len(out) == 0 {
		return nil, errors.New("at least one scope is required")
	}
	return out, nil
}

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !namePattern.MatchString(name) {
		return "", errors.New("name must be 1-63 chars: letters, digits, spaces, . _ -")
	}
	return name, nil
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Create(ctx context.Context, userID int64, name, prefix string, hash []byte, scopes []string, expiresAt *time.Time) (*Key, error) {
	var k Key
	err := r.pool.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, name, key_prefix, key_hash, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, key_prefix, scopes, created_at, expires_at, last_used_at, revoked_at`,
		userID, name, prefix, hash, scopes, expiresAt).
		Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.CreatedAt, &k.ExpiresAt, &k.LastUsedAt, &k.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *Repository) List(ctx context.Context, userID int64) ([]Key, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, name, key_prefix, scopes, created_at, expires_at, last_used_at, revoked_at
		FROM api_keys WHERE user_id = $1 ORDER BY id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Key
	for rows.Next() {
		var k Key
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.CreatedAt, &k.ExpiresAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *Repository) Revoke(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE api_keys SET revoked_at = now()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetPrincipal(ctx context.Context, hash []byte) (*Principal, error) {
	var p Principal
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, scopes, id FROM api_keys
		WHERE key_hash = $1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())`,
		hash).Scan(&p.UserID, &p.Scopes, &p.KeyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalid
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Touch(ctx context.Context, id int64) {
	_, _ = r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
}

type Service struct {
	repo *Repository
	log  *slog.Logger
}

func NewService(repo *Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Create issues a new key. The raw secret is returned exactly once.
func (s *Service) Create(ctx context.Context, userID int64, name string, scopes []string, expiresAt *time.Time) (*Created, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	scopes, err = ValidateScopes(scopes)
	if err != nil {
		return nil, err
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return nil, errors.New("expires_at must be in the future")
	}
	raw, err := newRawKey()
	if err != nil {
		return nil, err
	}
	k, err := s.repo.Create(ctx, userID, name, raw[:len(Prefix)+8], hashKey(raw), scopes, expiresAt)
	if err != nil {
		return nil, err
	}
	return &Created{Key: *k, Raw: raw}, nil
}

func (s *Service) List(ctx context.Context, userID int64) ([]Key, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Revoke(ctx context.Context, userID, id int64) error {
	return s.repo.Revoke(ctx, userID, id)
}

// Authenticate resolves a raw key to its principal, refreshing last_used_at
// asynchronously so the request path is not blocked.
func (s *Service) Authenticate(ctx context.Context, raw string) (*Principal, error) {
	if !strings.HasPrefix(raw, Prefix) {
		return nil, ErrInvalid
	}
	p, err := s.repo.GetPrincipal(ctx, hashKey(raw))
	if err != nil {
		return nil, err
	}
	go func() {
		bctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		s.repo.Touch(bctx, p.KeyID)
	}()
	return p, nil
}

// LooksLikeKey reports whether a bearer token has the API-key shape.
func LooksLikeKey(raw string) bool { return strings.HasPrefix(raw, Prefix) }
