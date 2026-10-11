package url

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a URL does not exist or is not owned by the caller.
var ErrNotFound = errors.New("url not found")

const urlColumns = `id, user_id, slug, original_url, title, favicon_url, is_active, click_count, expires_at, created_at, updated_at`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func scanURL(row pgx.Row) (*ShortURL, error) {
	var s ShortURL
	err := row.Scan(&s.ID, &s.UserID, &s.Slug, &s.OriginalURL, &s.Title, &s.FaviconURL, &s.IsActive,
		&s.ClickCount, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Insert creates a row. When onConflictIgnore is true a slug collision returns
// (nil, ErrSlugTaken) so the caller can retry with a fresh random slug.
func (r *Repository) Insert(ctx context.Context, userID *int64, slug, originalURL string, title, faviconURL *string, expiresAt *time.Time, onConflictIgnore bool) (*ShortURL, error) {
	var conflictClause string
	if onConflictIgnore {
		conflictClause = " ON CONFLICT (slug) DO NOTHING"
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO short_urls (user_id, slug, original_url, title, favicon_url, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`+conflictClause+`
		RETURNING `+urlColumns,
		userID, slug, originalURL, title, faviconURL, expiresAt)
	s, err := scanURL(row)
	if errors.Is(err, ErrNotFound) && onConflictIgnore {
		return nil, ErrSlugTaken
	}
	if isUniqueViolation(err) {
		return nil, ErrSlugTaken
	}
	return s, err
}

func (r *Repository) GetBySlug(ctx context.Context, slug string) (*ShortURL, error) {
	return scanURL(r.pool.QueryRow(ctx,
		`SELECT `+urlColumns+` FROM short_urls WHERE slug = $1`, slug))
}

func (r *Repository) GetByID(ctx context.Context, id int64, userID *int64) (*ShortURL, error) {
	return scanURL(r.pool.QueryRow(ctx,
		`SELECT `+urlColumns+` FROM short_urls WHERE id = $1 AND (user_id IS NOT DISTINCT FROM $2)`, id, userID))
}

type ListParams struct {
	UserID *int64
	Cursor *int64
	Limit  int
	Query  string
}

func (r *Repository) List(ctx context.Context, p ListParams) ([]ShortURL, *int64, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	q := `%` + p.Query + `%`
	rows, err := r.pool.Query(ctx, `
		SELECT `+urlColumns+` FROM short_urls
		WHERE user_id = $1
		  AND ($2::bigint IS NULL OR id < $2)
		  AND ($3 = '' OR slug ILIKE $3 OR original_url ILIKE $3 OR title ILIKE $3)
		ORDER BY id DESC
		LIMIT $4`,
		p.UserID, p.Cursor, q, p.Limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var items []ShortURL
	for rows.Next() {
		var s ShortURL
		if err := rows.Scan(&s.ID, &s.UserID, &s.Slug, &s.OriginalURL, &s.Title, &s.FaviconURL, &s.IsActive,
			&s.ClickCount, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, nil, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var next *int64
	if len(items) > p.Limit {
		items = items[:p.Limit]
		last := items[len(items)-1].ID
		next = &last
	}
	return items, next, nil
}

type UpdateFields struct {
	OriginalURL  *string
	Title        *string
	ClearTitle   bool
	IsActive     *bool
	ExpiresAt    *time.Time
	ClearExpires bool
}

func (r *Repository) Update(ctx context.Context, id int64, userID *int64, f UpdateFields) (*ShortURL, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE short_urls SET
			original_url = COALESCE($3, original_url),
			title = CASE WHEN $5 THEN NULL ELSE COALESCE($4, title) END,
			is_active = COALESCE($6, is_active),
			expires_at = CASE WHEN $7 THEN NULL ELSE COALESCE($8, expires_at) END,
			updated_at = now()
		WHERE id = $1 AND (user_id IS NOT DISTINCT FROM $2)
		RETURNING `+urlColumns,
		id, userID, f.OriginalURL, f.Title, f.ClearTitle, f.IsActive, f.ClearExpires, f.ExpiresAt)
	return scanURL(row)
}

func (r *Repository) Delete(ctx context.Context, id int64, userID *int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM short_urls WHERE id = $1 AND (user_id IS NOT DISTINCT FROM $2)`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Pool exposes the underlying pool for operational tasks (tests, admin).
func (r *Repository) Pool() *pgxpool.Pool { return r.pool }
