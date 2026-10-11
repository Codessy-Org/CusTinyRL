// Package integration contains env-gated tests against real Postgres and
// Redis. They are skipped unless TEST_DATABASE_URL is set. TEST_REDIS_ADDR
// defaults to localhost:6379 and uses a dedicated logical DB that is flushed
// per run.
package integration

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gauravkrrr/url-shortner/backend/internal/analytics"
	"github.com/gauravkrrr/url-shortner/backend/internal/apikey"
	"github.com/gauravkrrr/url-shortner/backend/internal/auth"
	"github.com/gauravkrrr/url-shortner/backend/internal/cache"
	"github.com/gauravkrrr/url-shortner/backend/internal/config"
	"github.com/gauravkrrr/url-shortner/backend/internal/db"
	"github.com/gauravkrrr/url-shortner/backend/internal/identity"
	"github.com/gauravkrrr/url-shortner/backend/internal/ratelimit"
	"github.com/gauravkrrr/url-shortner/backend/internal/redirect"
	"github.com/gauravkrrr/url-shortner/backend/internal/url"
)

type app struct {
	t         testing.TB
	urlSvc    *url.Service
	urlRepo   *url.Repository
	cache     *cache.Cache
	ingester  *analytics.Ingester
	querier   *analytics.Querier
	redirect  *redirect.Service
	limiter   *ratelimit.Limiter
	authSvc   *auth.Service
	apiKeySvc *apikey.Service
	cfg       *config.Config
	teardown  func()
}

func newApp(t testing.TB) *app {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Dedicated logical DB for tests; flush to start clean.
	rdb := cache.New(redisAddr, "", 9, 500*time.Millisecond, 200*time.Millisecond, time.Second, log)
	raw := redis.NewClient(&redis.Options{Addr: redisAddr, DB: 9})
	if err := raw.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush test redis: %v", err)
	}
	_ = raw.Close()

	limiter := ratelimit.New(redisAddr, "", 9, log)
	urlRepo := url.NewRepository(pool)
	urlSvc := url.NewService(urlRepo, rdb, 10, 5, "http://test.local", log)
	ingRepo := analytics.NewRepository(pool)
	ingester := analytics.NewIngester(ingRepo, 256, 16, 50*time.Millisecond, log)
	querier := analytics.NewQuerier(pool)
	redirectSvc := redirect.NewService(urlSvc, rdb, ingester, true, log)
	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, time.Hour, log)
	apiKeySvc := apikey.NewService(apikey.NewRepository(pool), log)
	authSvc.SetAPIKeyService(apiKeySvc)

	return &app{
		t: t, urlSvc: urlSvc, urlRepo: urlRepo, cache: rdb, ingester: ingester,
		querier: querier, redirect: redirectSvc, limiter: limiter, authSvc: authSvc,
		apiKeySvc: apiKeySvc,
		teardown:  func() { pool.Close() },
	}
}

func (a *app) clean() {
	ctx := context.Background()
	if _, err := a.urlRepo.Pool().Exec(ctx, "TRUNCATE short_urls, click_events, auth_tokens, api_keys, users RESTART IDENTITY CASCADE"); err != nil {
		a.t.Fatalf("truncate: %v", err)
	}
	raw := redis.NewClient(&redis.Options{Addr: testRedisAddr(), DB: 9})
	defer raw.Close()
	if err := raw.FlushDB(ctx).Err(); err != nil {
		a.t.Fatalf("flush test redis: %v", err)
	}
}

func TestCustomSlugLifecycle(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "lifecycle@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.urlSvc.Create(ctx, url.CreateInput{
		UserID:      &tok.User.ID,
		OriginalURL: "https://docs.google.com/document/d/example-document-id",
		CustomSlug:  ptr("gauravresumev1"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rec.Slug != "gauravresumev1" {
		t.Fatalf("slug = %q", rec.Slug)
	}

	// Duplicate → ErrSlugTaken.
	_, err = a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com", CustomSlug: ptr("gauravresumev1")})
	if !errors.Is(err, url.ErrSlugTaken) {
		t.Fatalf("expected ErrSlugTaken, got %v", err)
	}

	// Redirect resolves (cache miss → DB → populate).
	entry, outcome, err := a.redirect.Resolve(ctx, "gauravresumev1")
	if err != nil || outcome != redirect.OK {
		t.Fatalf("resolve: outcome=%v err=%v", outcome, err)
	}
	if entry.OriginalURL != "https://docs.google.com/document/d/example-document-id" {
		t.Fatalf("dest = %q", entry.OriginalURL)
	}

	// Cache hit path returns same data.
	entry2, outcome2, err := a.redirect.Resolve(ctx, "gauravresumev1")
	if err != nil || outcome2 != redirect.OK || entry2.ID != entry.ID {
		t.Fatalf("cached resolve mismatch")
	}

	// Update destination invalidates cache; next resolve sees new dest.
	newDest := "https://example.org/new-destination"
	if _, err := a.urlSvc.Update(ctx, rec.ID, url.UpdateInput{UserID: &tok.User.ID, OriginalURL: &newDest}); err != nil {
		t.Fatalf("update: %v", err)
	}
	entry3, outcome3, err := a.redirect.Resolve(ctx, "gauravresumev1")
	if err != nil || outcome3 != redirect.OK {
		t.Fatalf("resolve after update: %v %v", outcome3, err)
	}
	if entry3.OriginalURL != newDest {
		t.Fatalf("stale destination served: %q", entry3.OriginalURL)
	}

	// Disable → Inactive (even if cached).
	active := false
	if _, err := a.urlSvc.Update(ctx, rec.ID, url.UpdateInput{UserID: &tok.User.ID, IsActive: &active}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, outcome, _ := a.redirect.Resolve(ctx, "gauravresumev1"); outcome != redirect.Inactive {
		t.Fatalf("expected Inactive, got %v", outcome)
	}

	// Re-enable, then delete → NotFound.
	active = true
	if _, err := a.urlSvc.Update(ctx, rec.ID, url.UpdateInput{UserID: &tok.User.ID, IsActive: &active}); err != nil {
		t.Fatal(err)
	}
	if err := a.urlSvc.Delete(ctx, rec.ID, &tok.User.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, outcome, _ := a.redirect.Resolve(ctx, "gauravresumev1"); outcome != redirect.NotFound {
		t.Fatalf("expected NotFound after delete, got %v", outcome)
	}
}

func TestGeneratedSlugUniquenessUnderConcurrency(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	const n = 30
	var wg sync.WaitGroup
	errs := make(chan error, n)
	slugs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec, err := a.urlSvc.Create(ctx, url.CreateInput{OriginalURL: "https://example.com"})
			if err != nil {
				errs <- err
				return
			}
			slugs <- rec.Slug
		}()
	}
	wg.Wait()
	close(errs)
	close(slugs)
	for err := range errs {
		t.Fatalf("concurrent create failed: %v", err)
	}
	seen := map[string]bool{}
	for s := range slugs {
		if seen[s] {
			t.Fatalf("duplicate slug %q", s)
		}
		seen[s] = true
	}
	if len(seen) != n {
		t.Fatalf("got %d slugs, want %d", len(seen), n)
	}
}

func TestConcurrentSameCustomSlug(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "race@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successes, conflicts int
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com", CustomSlug: ptr("race-slug")})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, url.ErrSlugTaken):
				conflicts++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes != 1 || conflicts != n-1 {
		t.Fatalf("successes=%d conflicts=%d, want 1 and %d", successes, conflicts, n-1)
	}
}

func TestExpiredLink(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "expired@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com", CustomSlug: ptr("expiring")})
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Minute)
	if _, err := a.urlRepo.Pool().Exec(ctx,
		`UPDATE short_urls SET expires_at = $2 WHERE id = $1`, rec.ID, past); err != nil {
		t.Fatal(err)
	}
	// Bust any cache from create-time invalidation (none expected, but be safe).
	a.cache.Invalidate(ctx, "expiring")

	_, outcome, err := a.redirect.Resolve(ctx, "expiring")
	if err != nil {
		t.Fatal(err)
	}
	if outcome != redirect.Expired {
		t.Fatalf("outcome = %v, want Expired", outcome)
	}
}

func TestNegativeCacheProtectsDB(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	if _, outcome, _ := a.redirect.Resolve(ctx, "definitely-missing"); outcome != redirect.NotFound {
		t.Fatal("expected NotFound")
	}
	// Second lookup is served from the negative cache; the miss key must exist.
	raw := redis.NewClient(&redis.Options{Addr: testRedisAddr(), DB: 9})
	defer raw.Close()
	if n, err := raw.Exists(ctx, "url:miss:definitely-missing").Result(); err != nil || n != 1 {
		t.Fatalf("negative cache key missing (n=%d err=%v)", n, err)
	}
}

func TestPrefixSlug(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "prefix@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com", Prefix: ptr("resume")})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Slug) < 8 || rec.Slug[:7] != "resume-" {
		t.Fatalf("slug = %q", rec.Slug)
	}
}

func TestRateLimitAcrossInstances(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	ctx := context.Background()

	// Two limiters share the same Redis: models two API instances.
	other := ratelimit.New(testRedisAddr(), "", 9, nil)
	defer other.Close()
	policy := config.RateLimitPolicy{Name: "test_create", Limit: 5, Window: time.Minute}

	for i := 0; i < 5; i++ {
		res, err := a.limiter.Allow(ctx, policy, "user:1")
		if err != nil || !res.Allowed {
			t.Fatalf("request %d should be allowed: %+v err=%v", i+1, res, err)
		}
	}
	res, err := other.Allow(ctx, policy, "user:1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Fatal("6th request should be rejected by the shared limiter")
	}
	if res.RetryAfter <= 0 {
		t.Fatal("expected positive Retry-After")
	}
}

func TestAuthAndOwnership(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := a.authSvc.Register(ctx, "user@example.com", "password123"); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
	loggedIn, err := a.authSvc.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	u, err := a.authSvc.Authenticate(ctx, loggedIn.Token)
	if err != nil || u.ID != tok.User.ID {
		t.Fatalf("authenticate: %v", err)
	}

	uid := u.ID
	rec, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &uid, OriginalURL: "https://example.com", CustomSlug: ptr("owned")})
	if err != nil {
		t.Fatal(err)
	}
	// Another user cannot manage it.
	otherTok, err := a.authSvc.Register(ctx, "other@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	otherID := otherTok.User.ID
	if _, err := a.urlSvc.Get(ctx, rec.ID, &otherID); !errors.Is(err, url.ErrNotFound) {
		t.Fatalf("cross-user Get should fail, got %v", err)
	}
	if err := a.urlSvc.Delete(ctx, rec.ID, &otherID); !errors.Is(err, url.ErrNotFound) {
		t.Fatalf("cross-user Delete should fail, got %v", err)
	}
}

func TestAnalyticsIngestion(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "analytics@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com", CustomSlug: ptr("analyzed")})
	if err != nil {
		t.Fatal(err)
	}
	entry, outcome, err := a.redirect.Resolve(ctx, "analyzed")
	if err != nil || outcome != redirect.OK {
		t.Fatalf("resolve failed: %v %v", outcome, err)
	}
	for i := 0; i < 5; i++ {
		a.redirect.RecordClick(entry, "https://Twitter.com/some/status", "Mozilla/5.0 (iPhone)")
	}
	// Run a one-shot ingester loop by starting Run briefly.
	rctx, cancel := context.WithCancel(context.Background())
	go a.ingester.Run(rctx)
	time.Sleep(400 * time.Millisecond)
	cancel()
	a.ingester.Wait(time.Second)

	got, err := a.urlRepo.Pool().Query(ctx, `SELECT click_count FROM short_urls WHERE id = $1`, rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	var count int64
	if !got.Next() {
		t.Fatal("no row")
	}
	if err := got.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("click_count = %d, want 5 (events may need longer flush)", count)
	}

	summary, err := a.querier.Summary(ctx, rec.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalClicks != 5 {
		t.Fatalf("summary total = %d", summary.TotalClicks)
	}
	if len(summary.TopReferrers) == 0 || summary.TopReferrers[0].Label != "twitter.com" {
		t.Fatalf("referrers = %+v", summary.TopReferrers)
	}
}

func TestAPIKeyLifecycle(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "keys@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	uid := tok.User.ID

	created, err := a.apiKeySvc.Create(ctx, uid, "ci-pipeline", []string{"create"}, nil)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if created.Raw == "" || len(created.Raw) < 10 {
		t.Fatal("expected raw key")
	}
	if !apikey.LooksLikeKey(created.Raw) {
		t.Fatalf("raw key %q missing ck_ prefix", created.Raw)
	}

	// Authenticate resolves the same user.
	p, err := a.apiKeySvc.Authenticate(ctx, created.Raw)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if p.UserID != uid {
		t.Fatalf("principal user %d, want %d", p.UserID, uid)
	}
	if len(p.Scopes) != 1 || p.Scopes[0] != "create" {
		t.Fatalf("scopes = %v", p.Scopes)
	}

	// Session-token-shaped strings are rejected.
	if _, err := a.apiKeySvc.Authenticate(ctx, "not-a-key"); !errors.Is(err, apikey.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}

	// List shows prefix only, never the raw secret.
	keys, err := a.apiKeySvc.List(ctx, uid)
	if err != nil || len(keys) != 1 {
		t.Fatalf("list: %v len=%d", err, len(keys))
	}
	if keys[0].KeyPrefix == created.Raw {
		t.Fatal("full key must not be stored/listed")
	}

	// Revoke → subsequent authenticate fails.
	if err := a.apiKeySvc.Revoke(ctx, uid, created.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := a.apiKeySvc.Authenticate(ctx, created.Raw); !errors.Is(err, apikey.ErrInvalid) {
		t.Fatalf("revoked key should be invalid, got %v", err)
	}

	// Expired keys are rejected.
	exp, err := a.apiKeySvc.Create(ctx, uid, "short-lived", []string{"read", "manage"}, ptrTime(time.Now().Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	pool := a.urlRepo.Pool()
	if _, err := pool.Exec(ctx, `UPDATE api_keys SET expires_at = now() - interval '1 minute' WHERE id = $1`, exp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.apiKeySvc.Authenticate(ctx, exp.Raw); !errors.Is(err, apikey.ErrInvalid) {
		t.Fatalf("expired key should be invalid, got %v", err)
	}
}

func TestBotFilterSkipsAnalytics(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "bots@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com", CustomSlug: ptr("botfiltered")})
	if err != nil {
		t.Fatal(err)
	}
	entry, outcome, err := a.redirect.Resolve(ctx, "botfiltered")
	if err != nil || outcome != redirect.OK {
		t.Fatalf("resolve: %v %v", outcome, err)
	}
	// 3 human + 2 bot clicks; bots must be excluded.
	for i := 0; i < 3; i++ {
		a.redirect.RecordClick(entry, "", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	}
	for i := 0; i < 2; i++ {
		a.redirect.RecordClick(entry, "", "Mozilla/5.0 (compatible; Googlebot/2.1)")
		a.redirect.RecordClick(entry, "", "curl/8.4.0")
	}
	rctx, cancel := context.WithCancel(context.Background())
	go a.ingester.Run(rctx)
	time.Sleep(400 * time.Millisecond)
	cancel()
	a.ingester.Wait(time.Second)

	var count int64
	if err := a.urlRepo.Pool().QueryRow(ctx, `SELECT click_count FROM short_urls WHERE id = $1`, rec.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("click_count = %d, want 3 (bots excluded)", count)
	}
}

func TestPreviewStoresTitleAndFavicon(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "preview@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}

	// Stub fetcher: preview.Fetch's SSRF guard blocks loopback test servers,
	// so the HTTP layer is unit-tested in the preview package instead.
	a.urlSvc.SetPreview(func(ctx context.Context, dest string) (string, string, error) {
		return "Example Job Page", "https://cdn.example.com/fav.png", nil
	})

	rec, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com/jobs/123", CustomSlug: ptr("previewed")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rec.Title == nil || *rec.Title != "Example Job Page" {
		t.Fatalf("title = %v, want Example Job Page", rec.Title)
	}
	if rec.FaviconURL == nil || *rec.FaviconURL != "https://cdn.example.com/fav.png" {
		t.Fatalf("favicon = %v", rec.FaviconURL)
	}

	// Explicit title wins over the fetched one.
	rec2, err := a.urlSvc.Create(ctx, url.CreateInput{UserID: &tok.User.ID, OriginalURL: "https://example.com/jobs/123", Title: ptr("My Title"), CustomSlug: ptr("previewed2")})
	if err != nil {
		t.Fatal(err)
	}
	if rec2.Title == nil || *rec2.Title != "My Title" {
		t.Fatalf("explicit title lost: %v", rec2.Title)
	}
}

func TestRevokedKeyNeverFallsBackToAnonymous(t *testing.T) {
	a := newApp(t)
	defer a.teardown()
	a.clean()
	ctx := context.Background()

	tok, err := a.authSvc.Register(ctx, "revoked@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	created, err := a.apiKeySvc.Create(ctx, tok.User.ID, "temp", []string{"create"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Mini router mirroring POST /urls: optionalAuth + create handler that
	// records whether the request arrived authenticated or anonymous.
	var sawUserID *int64
	h := a.authSvc.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawUserID = identity.UserID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer "+created.Raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("valid key status = %d", resp.StatusCode)
	}
	if sawUserID == nil || *sawUserID != tok.User.ID {
		t.Fatalf("request not attributed to key owner")
	}

	if err := a.apiKeySvc.Revoke(ctx, tok.User.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	req2, _ := http.NewRequest(http.MethodPost, srv.URL, nil)
	req2.Header.Set("Authorization", "Bearer "+created.Raw)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked key status = %d, want 401 (must not degrade to anonymous)", resp2.StatusCode)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func testRedisAddr() string {
	if v := os.Getenv("TEST_REDIS_ADDR"); v != "" {
		return v
	}
	return "localhost:6379"
}

func ptr(s string) *string { return &s }
