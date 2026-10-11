package url

type CreateURLRequest struct {
	OriginalURL string  `json:"original_url"`
	CustomSlug  *string `json:"custom_slug"`
	Prefix      *string `json:"prefix"`
	Title       *string `json:"title"`
	ExpiresAt   *string `json:"expires_at"`
}

type UpdateURLRequest struct {
	OriginalURL *string `json:"original_url"`
	IsActive    *bool   `json:"is_active"`
	Title       *string `json:"title"`
	ExpiresAt   *string `json:"expires_at"`
	ClearExpiry *bool   `json:"clear_expires_at"`
}

type URLResponse struct {
	ID          int64   `json:"id"`
	Slug        string  `json:"slug"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	Title       *string `json:"title"`
	FaviconURL  *string `json:"favicon_url"`
	IsActive    bool    `json:"is_active"`
	IsExpired   bool    `json:"is_expired"`
	ClickCount  int64   `json:"click_count"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
}

type URLListResponse struct {
	Data       []URLResponse `json:"data"`
	NextCursor *string       `json:"next_cursor"`
}
