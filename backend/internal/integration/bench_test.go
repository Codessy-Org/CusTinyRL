package integration

import (
	"context"
	"testing"

	"github.com/gauravkrrr/url-shortner/backend/internal/redirect"
	"github.com/gauravkrrr/url-shortner/backend/internal/url"
)

// Benchmarks measure real cache-hit and cache-miss redirect resolution plus
// concurrent URL creation against live Postgres + Redis (TEST_DATABASE_URL
// required). Results are printed by `go test -bench`; nothing is hardcoded.
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/integration -bench . -benchtime 2s

func BenchmarkRedirectCacheHit(b *testing.B) {
	a := newApp(b)
	defer a.teardown()
	a.clean()
	ctx := context.Background()
	if _, err := a.urlSvc.Create(ctx, url.CreateInput{OriginalURL: "https://example.com", CustomSlug: strPtr("benchhit")}); err != nil {
		b.Fatal(err)
	}
	if _, outcome, err := a.redirect.Resolve(ctx, "benchhit"); err != nil || outcome != redirect.OK {
		b.Fatalf("warmup failed: %v %v", outcome, err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, outcome, err := a.redirect.Resolve(ctx, "benchhit"); err != nil || outcome != redirect.OK {
				b.Fatalf("resolve: %v %v", outcome, err)
			}
		}
	})
}

func BenchmarkRedirectCacheMiss(b *testing.B) {
	a := newApp(b)
	defer a.teardown()
	a.clean()
	ctx := context.Background()
	if _, err := a.urlSvc.Create(ctx, url.CreateInput{OriginalURL: "https://example.com", CustomSlug: strPtr("benchmiss")}); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		a.cache.Invalidate(ctx, "benchmiss")
		b.StartTimer()
		if _, outcome, err := a.redirect.Resolve(ctx, "benchmiss"); err != nil || outcome != redirect.OK {
			b.Fatalf("resolve: %v %v", outcome, err)
		}
	}
}

func BenchmarkCreateAutoSlug(b *testing.B) {
	a := newApp(b)
	defer a.teardown()
	a.clean()
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := a.urlSvc.Create(ctx, url.CreateInput{OriginalURL: "https://example.com"}); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func strPtr(s string) *string { return &s }
