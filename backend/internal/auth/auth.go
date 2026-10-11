// Package auth provides minimal email+password authentication issuing opaque
// bearer tokens. Tokens are stored only as SHA-256 digests; bcrypt cost keeps
// stolen password hashes expensive to attack.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/gauravkrrr/url-shortner/backend/internal/apikey"
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Token struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidToken       = errors.New("invalid token")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at`, email, passwordHash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return User{}, ErrEmailTaken
	}
	return u, err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, created_at, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return User{}, "", err
	}
	out := ""
	if hash != nil {
		out = *hash
	}
	return u, out, err
}

// CreateUserWithGoogle links or creates a user for a Google identity. Google
// emails are verified, so an existing account with the same email is linked
// (password login keeps working) rather than duplicated.
func (r *Repository) UpsertGoogleUser(ctx context.Context, sub, email, name, avatarURL string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, google_sub, name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (google_sub) DO UPDATE
			SET name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
		RETURNING id, email, created_at`, email, sub, name, avatarURL).
		Scan(&u.ID, &u.Email, &u.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		// Email belongs to another account (e.g. registered with a password):
		// link the Google identity to it instead.
		linkErr := r.pool.QueryRow(ctx, `
			UPDATE users SET google_sub = $2, name = $3, avatar_url = $4
			WHERE email = $1
			RETURNING id, email, created_at`, email, sub, name, avatarURL).
			Scan(&u.ID, &u.Email, &u.CreatedAt)
		if linkErr != nil {
			return User{}, ErrEmailTaken
		}
		return u, nil
	}
	return u, err
}

func (r *Repository) CreateToken(ctx context.Context, userID int64, hash []byte, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, expiresAt)
	return err
}

func (r *Repository) GetUserByTokenHash(ctx context.Context, hash []byte) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.created_at
		FROM auth_tokens t JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1 AND t.expires_at > now()`, hash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidToken
	}
	return u, err
}

func (r *Repository) DeleteToken(ctx context.Context, hash []byte) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM auth_tokens WHERE token_hash = $1`, hash)
	return err
}

type Service struct {
	repo     *Repository
	tokenTTL time.Duration
	log      *slog.Logger
	keys     *apikey.Service // optional; enables API-key credentials
	google   *GoogleProvider // optional; enables Google OAuth login
}

func NewService(repo *Repository, tokenTTL time.Duration, log *slog.Logger) *Service {
	return &Service{repo: repo, tokenTTL: tokenTTL, log: log}
}

// SetGoogleProvider enables "Continue with Google" when configured.
func (s *Service) SetGoogleProvider(g *GoogleProvider) { s.google = g }

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func newTokenString() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func normalizeEmail(email string) string {
	// Case-insensitive uniqueness without a functional index migration.
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) Register(ctx context.Context, email, password string) (*Token, error) {
	email = normalizeEmail(email)
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		return nil, err
	}
	return s.issueToken(ctx, u)
}

func (s *Service) Login(ctx context.Context, email, password string) (*Token, error) {
	email = normalizeEmail(email)
	u, storedHash, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueToken(ctx, u)
}

func (s *Service) issueToken(ctx context.Context, u User) (*Token, error) {
	raw, err := newTokenString()
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(s.tokenTTL)
	if err := s.repo.CreateToken(ctx, u.ID, hashToken(raw), expires); err != nil {
		return nil, err
	}
	return &Token{Token: raw, ExpiresAt: expires, User: u}, nil
}

func (s *Service) Authenticate(ctx context.Context, raw string) (User, error) {
	if raw == "" {
		return User{}, ErrInvalidToken
	}
	return s.repo.GetUserByTokenHash(ctx, hashToken(raw))
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.repo.DeleteToken(ctx, hashToken(raw))
}

func validateEmail(email string) error {
	if len(email) < 3 || len(email) > 254 {
		return errors.New("invalid email address")
	}
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 || at == len(email)-1 {
		return errors.New("invalid email address")
	}
	return nil
}
