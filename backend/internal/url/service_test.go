package url

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

type fakeStore struct {
	mu   sync.Mutex
	slug map[string]*ShortURL
	next int64
	fail int // number of Insert calls that report slug conflicts before succeeding
}

func newFakeStore() *fakeStore {
	return &fakeStore{slug: map[string]*ShortURL{}, next: 1}
}

func (f *fakeStore) Insert(ctx context.Context, userID *int64, slug, originalURL string, title, faviconURL *string, expiresAt *time.Time, onConflictIgnore bool) (*ShortURL, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail > 0 {
		f.fail--
		if onConflictIgnore {
			return nil, ErrSlugTaken
		}
	}
	if _, exists := f.slug[slug]; exists {
		if onConflictIgnore {
			return nil, ErrSlugTaken
		}
		return nil, ErrSlugTaken
	}
	id := f.next
	f.next++
	rec := &ShortURL{ID: id, UserID: userID, Slug: slug, OriginalURL: originalURL, Title: title, FaviconURL: faviconURL, IsActive: true, ExpiresAt: expiresAt, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.slug[slug] = rec
	cp := *rec
	return &cp, nil
}

func (f *fakeStore) GetBySlug(ctx context.Context, slug string) (*ShortURL, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec, ok := f.slug[slug]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *rec
	return &cp, nil
}

func (f *fakeStore) GetByID(ctx context.Context, id int64, userID *int64) (*ShortURL, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, rec := range f.slug {
		if rec.ID == id {
			cp := *rec
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeStore) List(ctx context.Context, p ListParams) ([]ShortURL, *int64, error) {
	return nil, nil, nil
}

func (f *fakeStore) Update(ctx context.Context, id int64, userID *int64, uf UpdateFields) (*ShortURL, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, rec := range f.slug {
		if rec.ID == id {
			if uf.OriginalURL != nil {
				rec.OriginalURL = *uf.OriginalURL
			}
			if uf.IsActive != nil {
				rec.IsActive = *uf.IsActive
			}
			cp := *rec
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeStore) Delete(ctx context.Context, id int64, userID *int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for slug, rec := range f.slug {
		if rec.ID == id {
			delete(f.slug, slug)
			return nil
		}
	}
	return ErrNotFound
}

type fakeInvalidator struct {
	mu    sync.Mutex
	slugs []string
}

func (f *fakeInvalidator) Invalidate(ctx context.Context, slug string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.slugs = append(f.slugs, slug)
}

func testService(t *testing.T, store Store) (*Service, *fakeInvalidator) {
	t.Helper()
	inv := &fakeInvalidator{}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewService(store, inv, 10, 5, "https://sh.example", log), inv
}

func TestCreateCustomSlug(t *testing.T) {
	store := newFakeStore()
	svc, inv := testService(t, store)

	rec, err := svc.Create(context.Background(), CreateInput{
		UserID:      int64Ptr(1),
		OriginalURL: "https://docs.google.com/document/d/example-document-id",
		CustomSlug:  strPtr("gauravresumev1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Slug != "gauravresumev1" {
		t.Fatalf("slug = %q", rec.Slug)
	}
	if len(inv.slugs) == 0 {
		t.Fatal("expected cache invalidation after create")
	}
}

func TestCreateCustomSlugTaken(t *testing.T) {
	store := newFakeStore()
	svc, _ := testService(t, store)

	in := CreateInput{UserID: int64Ptr(1), OriginalURL: "https://example.com", CustomSlug: strPtr("taken")}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(context.Background(), in)
	if !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("expected ErrSlugTaken, got %v", err)
	}
}

func TestCreateCustomSlugPrefixIgnored(t *testing.T) {
	store := newFakeStore()
	svc, _ := testService(t, store)
	rec, err := svc.Create(context.Background(), CreateInput{
		UserID:      int64Ptr(1),
		OriginalURL: "https://example.com",
		CustomSlug:  strPtr("exact-slug"),
		Prefix:      strPtr("resume"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Slug != "exact-slug" {
		t.Fatalf("prefix must not modify custom slug, got %q", rec.Slug)
	}
}

func TestCreateAnonymousCustomSlugForbidden(t *testing.T) {
	store := newFakeStore()
	svc, _ := testService(t, store)
	_, err := svc.Create(context.Background(), CreateInput{
		OriginalURL: "https://example.com",
		CustomSlug:  strPtr("vanity-slug"),
	})
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("expected ErrAuthRequired for anonymous custom slug, got %v", err)
	}
}

func TestCreateAnonymousPrefixForbidden(t *testing.T) {
	store := newFakeStore()
	svc, _ := testService(t, store)
	_, err := svc.Create(context.Background(), CreateInput{
		OriginalURL: "https://example.com",
		Prefix:      strPtr("resume"),
	})
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("expected ErrAuthRequired for anonymous prefix, got %v", err)
	}
	// Plain anonymous auto-create stays allowed.
	rec, err := svc.Create(context.Background(), CreateInput{OriginalURL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.UserID != nil {
		t.Fatal("anonymous link must not have a user id")
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestCreateAutoSlugRetriesOnConflict(t *testing.T) {
	store := newFakeStore()
	store.fail = 2 // first two generated inserts collide
	svc, _ := testService(t, store)
	rec, err := svc.Create(context.Background(), CreateInput{OriginalURL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Slug) != 10 {
		t.Fatalf("slug = %q", rec.Slug)
	}
}

func TestCreateAutoSlugRetriesExhausted(t *testing.T) {
	store := newFakeStore()
	store.fail = 100
	svc, _ := testService(t, store)
	_, err := svc.Create(context.Background(), CreateInput{OriginalURL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestCreateWithPrefix(t *testing.T) {
	store := newFakeStore()
	svc, _ := testService(t, store)
	rec, err := svc.Create(context.Background(), CreateInput{
		UserID:      int64Ptr(1),
		OriginalURL: "https://example.com",
		Prefix:      strPtr("resume"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Slug[:7] != "resume-" {
		t.Fatalf("slug = %q", rec.Slug)
	}
}

func TestCreateRejectsBadURL(t *testing.T) {
	store := newFakeStore()
	svc, _ := testService(t, store)
	_, err := svc.Create(context.Background(), CreateInput{OriginalURL: "javascript:alert(1)"})
	var verr ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestUpdateInvalidatesCache(t *testing.T) {
	store := newFakeStore()
	svc, inv := testService(t, store)
	rec, err := svc.Create(context.Background(), CreateInput{UserID: int64Ptr(1), OriginalURL: "https://a.example", CustomSlug: strPtr("link1")})
	if err != nil {
		t.Fatal(err)
	}
	active := false
	if _, err := svc.Update(context.Background(), rec.ID, UpdateInput{IsActive: &active}); err != nil {
		t.Fatal(err)
	}
	if inv.slugs[len(inv.slugs)-1] != "link1" {
		t.Fatal("expected invalidation of updated slug")
	}
}

func TestShortURLActive(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	cases := []struct {
		name string
		s    ShortURL
		want bool
	}{
		{"active", ShortURL{IsActive: true}, true},
		{"disabled", ShortURL{IsActive: false}, false},
		{"expired", ShortURL{IsActive: true, ExpiresAt: &past}, false},
		{"future expiry", ShortURL{IsActive: true, ExpiresAt: &future}, true},
	}
	for _, c := range cases {
		if got := c.s.Active(now); got != c.want {
			t.Fatalf("%s: Active() = %v, want %v", c.name, got, c.want)
		}
	}
}

func strPtr(s string) *string { return &s }
