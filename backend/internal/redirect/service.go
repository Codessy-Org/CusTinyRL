// Package redirect resolves public short links through the Redis cache with
// Postgres fallback, and records click analytics asynchronously.
package redirect

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gauravkrrr/url-shortner/backend/internal/analytics"
	"github.com/gauravkrrr/url-shortner/backend/internal/cache"
	"github.com/gauravkrrr/url-shortner/backend/internal/metrics"
	"github.com/gauravkrrr/url-shortner/backend/internal/url"
)

type Outcome int

const (
	OK Outcome = iota
	NotFound
	Inactive
	Expired
)

type Service struct {
	urls      *url.Service
	cache     *cache.Cache
	ingester  *analytics.Ingester
	botFilter bool
	log       *slog.Logger
}

func NewService(urls *url.Service, c *cache.Cache, ingester *analytics.Ingester, botFilter bool, log *slog.Logger) *Service {
	return &Service{urls: urls, cache: c, ingester: ingester, botFilter: botFilter, log: log}
}

// Resolve looks up the slug. The returned entry is only valid for Outcome==OK.
func (s *Service) Resolve(ctx context.Context, slug string) (*cache.Entry, Outcome, error) {
	entry, found, err := s.cache.GetOrLoad(ctx, slug, func(lctx context.Context) (*cache.Entry, bool, error) {
		rec, err := s.urls.ResolveBySlug(lctx, slug)
		if errors.Is(err, url.ErrNotFound) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		return &cache.Entry{
			ID:          rec.ID,
			Slug:        rec.Slug,
			OriginalURL: rec.OriginalURL,
			Title:       rec.Title,
			IsActive:    rec.IsActive,
			ExpiresAt:   rec.ExpiresAt,
		}, true, nil
	})
	if err != nil {
		return nil, NotFound, err
	}
	if !found || entry == nil {
		return nil, NotFound, nil
	}

	now := time.Now()
	if !entry.IsActive {
		// Disabled links stay cached as inactive so a stale cache can never
		// redirect them, even before invalidation lands.
		return entry, Inactive, nil
	}
	if entry.ExpiresAt != nil && !entry.ExpiresAt.After(now) {
		return entry, Expired, nil
	}
	return entry, OK, nil
}

// RecordClick enqueues a privacy-conscious click event (host-only referrer,
// coarse device category, no IP). When bot filtering is enabled, crawlers,
// monitors, and scripts still redirect but are excluded from analytics.
func (s *Service) RecordClick(entry *cache.Entry, referrer, userAgent string) {
	if s.ingester == nil {
		return
	}
	if s.botFilter && analytics.IsBot(userAgent) {
		metrics.Registry.ClicksBotFiltered.Inc()
		return
	}
	accepted := s.ingester.Record(analytics.Event{
		ShortURLID: entry.ID,
		ClickedAt:  time.Now().UTC(),
		Referrer:   analytics.SanitizeReferrer(referrer),
		Device:     analytics.ClassifyDevice(userAgent),
	}, metrics.Registry.AnalyticsDropped.Inc)
	if accepted {
		metrics.Registry.AnalyticsEnqueued.Inc()
	}
}
