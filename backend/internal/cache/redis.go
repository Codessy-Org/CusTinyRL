// Package cache implements a Redis-backed resolution cache. Redis is never
// authoritative: every write path invalidates keys, and expired/disabled state
// is re-checked from the cached record on every read.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"github.com/gauravkrrr/url-shortner/backend/internal/metrics"
)

// Entry is the cached projection required to safely resolve a redirect.
type Entry struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	OriginalURL string     `json:"original_url"`
	Title       *string    `json:"title,omitempty"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Active reports whether the cached record should currently redirect.
func (e *Entry) Active(now time.Time) bool {
	if !e.IsActive {
		return false
	}
	if e.ExpiresAt != nil && !e.ExpiresAt.After(now) {
		return false
	}
	return true
}

type Cache struct {
	client  *redis.Client
	ttl     time.Duration
	negTTL  time.Duration
	timeout time.Duration
	log     *slog.Logger
	group   singleflight.Group
}

func New(addr, password string, db int, ttl, negTTL, timeout time.Duration, log *slog.Logger) *Cache {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	return &Cache{client: client, ttl: ttl, negTTL: negTTL, timeout: timeout, log: log}
}

func key(slug string) string    { return "url:" + slug }
func negKey(slug string) string { return "url:miss:" + slug }

// Ping verifies Redis connectivity (readiness probe).
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Close() error { return c.client.Close() }

// GetOrLoad returns the cached entry, coalescing concurrent misses for the
// same slug (stampede protection). found=false means either a negative-cache
// hit or a verified absence (entry=nil).
func (c *Cache) GetOrLoad(ctx context.Context, slug string, load func(context.Context) (*Entry, bool, error)) (*Entry, bool, error) {
	tctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Fast path: positive lookup.
	val, err := c.client.Get(tctx, key(slug)).Result()
	if err == nil {
		var e Entry
		if jerr := json.Unmarshal([]byte(val), &e); jerr == nil {
			metrics.Registry.CacheHits.Inc()
			return &e, true, nil
		}
		// Corrupt entry: drop and fall through to the database.
		_ = c.client.Del(context.WithoutCancel(tctx), key(slug)).Err()
	} else if !errors.Is(err, redis.Nil) {
		// Redis unavailable: degrade gracefully to the database.
		c.noteErr("get", slug, err)
	}

	// Negative cache.
	if n, nerr := c.client.Get(tctx, negKey(slug)).Result(); nerr == nil && n == "1" {
		metrics.Registry.CacheNegativeHits.Inc()
		return nil, false, nil
	}

	// Coalesce DB loads for the same slug.
	v, err, _ := c.group.Do(slug, func() (any, error) {
		// Re-check cache inside the flight (another flight may have filled it).
		if val, ferr := c.client.Get(context.WithoutCancel(tctx), key(slug)).Result(); ferr == nil {
			var e Entry
			if jerr := json.Unmarshal([]byte(val), &e); jerr == nil {
				return &e, nil
			}
		}
		// Detach from the leader request's context so one canceled caller
		// cannot poison concurrent waiters; bound the load with its own timeout.
		lctx, lcancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer lcancel()
		metrics.Registry.CacheMisses.Inc()
		entry, found, lerr := load(lctx)
		if lerr != nil {
			return nil, lerr
		}
		if found && entry != nil {
			c.Set(context.WithoutCancel(ctx), entry)
			return entry, nil
		}
		c.SetNegative(context.WithoutCancel(ctx), slug)
		return (*Entry)(nil), nil
	})
	if err != nil {
		return nil, false, err
	}
	entry, _ := v.(*Entry)
	return entry, entry != nil, nil
}

// Set caches a valid resolution entry.
func (c *Cache) Set(ctx context.Context, e *Entry) {
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	ttl := c.ttl
	if e.ExpiresAt != nil {
		if remaining := time.Until(*e.ExpiresAt); remaining < ttl {
			if remaining <= 0 {
				return
			}
			ttl = remaining
		}
	}
	if err := c.client.Set(ctx, key(e.Slug), b, ttl).Err(); err != nil {
		c.noteErr("set", e.Slug, err)
	}
}

// SetNegative briefly caches a missing slug to protect Postgres from
// repeated lookups of nonexistent slugs.
func (c *Cache) SetNegative(ctx context.Context, slug string) {
	if err := c.client.Set(ctx, negKey(slug), "1", c.negTTL).Err(); err != nil {
		c.noteErr("set_negative", slug, err)
	}
}

// Invalidate drops positive and negative entries for a slug. Called on every
// create/update/delete so stale destinations are never served.
func (c *Cache) Invalidate(ctx context.Context, slug string) {
	if err := c.client.Del(ctx, key(slug), negKey(slug)).Err(); err != nil {
		c.noteErr("invalidate", slug, err)
	}
}

func (c *Cache) noteErr(op, slug string, err error) {
	metrics.Registry.CacheErrors.Inc()
	if c.log != nil {
		c.log.Warn("redis cache error", "op", op, "slug", slug, "err", fmt.Sprint(err))
	}
}
