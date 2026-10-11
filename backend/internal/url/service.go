package url

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Sentinel service errors mapped to HTTP responses by the handler.
var (
	ErrInvalidInput = errors.New("invalid input")
)

type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Invalidator lets the service drop stale cache entries without importing a
// concrete cache implementation.
type Invalidator interface {
	Invalidate(ctx context.Context, slug string)
}

// Store abstracts persistence so the service can be unit tested.
type Store interface {
	Insert(ctx context.Context, userID *int64, slug, originalURL string, title, faviconURL *string, expiresAt *time.Time, onConflictIgnore bool) (*ShortURL, error)
	GetBySlug(ctx context.Context, slug string) (*ShortURL, error)
	GetByID(ctx context.Context, id int64, userID *int64) (*ShortURL, error)
	List(ctx context.Context, p ListParams) ([]ShortURL, *int64, error)
	Update(ctx context.Context, id int64, userID *int64, f UpdateFields) (*ShortURL, error)
	Delete(ctx context.Context, id int64, userID *int64) error
}

// PreviewFetcher retrieves page metadata for a destination. It is optional;
// nil disables preview fetching.
type PreviewFetcher func(ctx context.Context, rawURL string) (title, faviconURL string, err error)

type Service struct {
	repo       Store
	cache      Invalidator
	defaultLen int
	maxRetries int
	baseURL    string
	preview    PreviewFetcher
	log        *slog.Logger
}

func NewService(repo Store, cache Invalidator, defaultSlugLen, maxRetries int, baseURL string, log *slog.Logger) *Service {
	if maxRetries <= 0 {
		maxRetries = 5
	}
	return &Service{repo: repo, cache: cache, defaultLen: defaultSlugLen, maxRetries: maxRetries, baseURL: baseURL, log: log}
}

// SetPreview enables destination metadata fetching (title + favicon).
func (s *Service) SetPreview(f PreviewFetcher) { s.preview = f }

type CreateInput struct {
	UserID      *int64
	OriginalURL string
	CustomSlug  *string
	Prefix      *string
	Title       *string
	ExpiresAt   *time.Time
}

// fetchPreview fills title/favicon from the destination page when preview
// fetching is enabled. Best-effort: failures leave both empty.
func (s *Service) fetchPreview(ctx context.Context, dest string, explicitTitle *string) (*string, *string) {
	if s.preview == nil {
		return explicitTitle, nil
	}
	title, favicon, err := s.preview(ctx, dest)
	if err != nil {
		if s.log != nil {
			s.log.Debug("preview fetch failed", "err", err.Error())
		}
		return explicitTitle, nil
	}
	outTitle := explicitTitle
	if (outTitle == nil || *outTitle == "") && title != "" {
		outTitle = &title
	}
	var fav *string
	if favicon != "" {
		fav = &favicon
	}
	return outTitle, fav
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*ShortURL, error) {
	dest, err := ValidateDestinationURL(in.OriginalURL)
	if err != nil {
		return nil, ValidationError{Msg: err.Error()}
	}

	var title *string
	if in.Title != nil && *in.Title != "" {
		t := *in.Title
		if len(t) > 200 {
			return nil, ValidationError{Msg: "title must be at most 200 characters"}
		}
		title = &t
	}
	title, favicon := s.fetchPreview(ctx, dest, title)

	if in.CustomSlug != nil && *in.CustomSlug != "" {
		// Vanity slugs are a claim on a name — only signed-in users can hold
		// them (anonymous links are unmanageable and rate-limited).
		if in.UserID == nil {
			return nil, ErrAuthRequired
		}
		// Custom-slug mode: prefixes never modify user-chosen slugs.
		slug, err := ValidateCustomSlug(*in.CustomSlug)
		if err != nil {
			return nil, ValidationError{Msg: err.Error()}
		}
		rec, err := s.repo.Insert(ctx, in.UserID, slug, dest, title, favicon, in.ExpiresAt, false)
		if errors.Is(err, ErrSlugTaken) {
			return nil, ErrSlugTaken
		}
		if err != nil {
			return nil, err
		}
		s.invalidate(rec.Slug)
		return rec, nil
	}

	// Automatic mode with optional prefix.
	if in.Prefix != nil && *in.Prefix != "" {
		if in.UserID == nil {
			return nil, ErrAuthRequired
		}
		prefix, err := ValidatePrefix(*in.Prefix)
		if err != nil {
			return nil, ValidationError{Msg: err.Error()}
		}
		for attempt := 0; attempt < s.maxRetries; attempt++ {
			slug, err := GenerateWithPrefix(prefix, s.defaultLen)
			if err != nil {
				return nil, err
			}
			rec, err := s.repo.Insert(ctx, in.UserID, slug, dest, title, favicon, in.ExpiresAt, true)
			if errors.Is(err, ErrSlugTaken) {
				continue
			}
			if err != nil {
				return nil, err
			}
			s.invalidate(rec.Slug)
			return rec, nil
		}
		return nil, errors.New("could not allocate a unique slug, try again")
	}

	for attempt := 0; attempt < s.maxRetries; attempt++ {
		slug, err := GenerateSlug(s.defaultLen)
		if err != nil {
			return nil, err
		}
		rec, err := s.repo.Insert(ctx, in.UserID, slug, dest, title, favicon, in.ExpiresAt, true)
		if errors.Is(err, ErrSlugTaken) {
			continue
		}
		if err != nil {
			return nil, err
		}
		s.invalidate(rec.Slug)
		return rec, nil
	}
	return nil, errors.New("could not allocate a unique slug, try again")
}

func (s *Service) Get(ctx context.Context, id int64, userID *int64) (*ShortURL, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) List(ctx context.Context, userID *int64, cursor *int64, limit int, query string) ([]ShortURL, *int64, error) {
	return s.repo.List(ctx, ListParams{UserID: userID, Cursor: cursor, Limit: limit, Query: query})
}

type UpdateInput struct {
	UserID      *int64
	OriginalURL *string
	IsActive    *bool
	Title       *string
	ExpiresAt   *time.Time
	ClearExpiry bool
	ClearTitle  bool
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*ShortURL, error) {
	fields := UpdateFields{
		OriginalURL:  in.OriginalURL,
		Title:        in.Title,
		ClearTitle:   in.ClearTitle,
		IsActive:     in.IsActive,
		ExpiresAt:    in.ExpiresAt,
		ClearExpires: in.ClearExpiry,
	}
	if in.OriginalURL != nil {
		dest, err := ValidateDestinationURL(*in.OriginalURL)
		if err != nil {
			return nil, ValidationError{Msg: err.Error()}
		}
		fields.OriginalURL = &dest
	}
	rec, err := s.repo.Update(ctx, id, in.UserID, fields)
	if err != nil {
		return nil, err
	}
	s.invalidate(rec.Slug)
	return rec, nil
}

func (s *Service) Delete(ctx context.Context, id int64, userID *int64) error {
	rec, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.invalidate(rec.Slug)
	return nil
}

// ResolveBySlug is used by the redirect path; it bypasses ownership checks.
func (s *Service) ResolveBySlug(ctx context.Context, slug string) (*ShortURL, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *Service) BaseURL() string { return s.baseURL }

func (s *Service) invalidate(slug string) {
	if s.cache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.cache.Invalidate(ctx, slug)
}
