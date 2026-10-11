package url

import "time"

type ShortURL struct {
	ID          int64      `json:"id"`
	UserID      *int64     `json:"user_id"`
	Slug        string     `json:"slug"`
	OriginalURL string     `json:"original_url"`
	Title       *string    `json:"title"`
	FaviconURL  *string    `json:"favicon_url"`
	IsActive    bool       `json:"is_active"`
	ClickCount  int64      `json:"click_count"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Active reports whether the link should currently redirect.
func (s *ShortURL) Active(now time.Time) bool {
	if !s.IsActive {
		return false
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(now) {
		return false
	}
	return true
}
