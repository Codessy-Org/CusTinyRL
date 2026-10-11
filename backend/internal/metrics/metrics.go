// Package metrics provides lightweight in-process instrumentation exposed as
// JSON on GET /metrics. It is intentionally dependency-free; a Prometheus
// exporter can be layered on later.
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Counter struct{ v atomic.Int64 }

func (c *Counter) Inc()         { c.v.Add(1) }
func (c *Counter) Add(n int64)  { c.v.Add(n) }
func (c *Counter) Value() int64 { return c.v.Load() }

type durationHist struct {
	mu     sync.Mutex
	counts []int64
	bounds []time.Duration
	total  int64
	sumNS  int64
}

func newDurationHist(boundsMs []int64) *durationHist {
	h := &durationHist{counts: make([]int64, len(boundsMs))}
	for _, b := range boundsMs {
		h.bounds = append(h.bounds, time.Duration(b)*time.Millisecond)
	}
	return h
}

func (h *durationHist) Observe(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.total++
	h.sumNS += int64(d)
	for i, b := range h.bounds {
		if d <= b {
			h.counts[i]++
			return
		}
	}
}

func (h *durationHist) snapshot() map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	buckets := make([]map[string]int64, 0, len(h.bounds))
	var cum int64
	for i, b := range h.bounds {
		cum += h.counts[i]
		buckets = append(buckets, map[string]int64{"le_ms": b.Milliseconds(), "count": cum})
	}
	var avg float64
	if h.total > 0 {
		avg = float64(h.sumNS) / float64(h.total) / 1e6
	}
	return map[string]any{
		"buckets":  buckets,
		"count":    h.total,
		"avg_ms":   avg,
		"overflow": h.total - cum,
	}
}

// Registry holds all application metrics.
var Registry = struct {
	RedirectTotal       *Counter
	RedirectErrors      *Counter
	RedirectLatency     *durationHist
	CacheHits           *Counter
	CacheMisses         *Counter
	CacheNegativeHits   *Counter
	CacheErrors         *Counter
	RateLimitRejected   sync.Map // policy -> *Counter
	AnalyticsEnqueued   *Counter
	AnalyticsDropped    *Counter
	AnalyticsFlushed    *Counter
	AnalyticsFailures   *Counter
	ClicksBotFiltered   *Counter
	URLsCreated         *Counter
	URLsCreatedRejected *Counter
}{
	RedirectTotal:       &Counter{},
	RedirectErrors:      &Counter{},
	RedirectLatency:     newDurationHist([]int64{5, 10, 25, 50, 100, 250, 500, 1000}),
	CacheHits:           &Counter{},
	CacheMisses:         &Counter{},
	CacheNegativeHits:   &Counter{},
	CacheErrors:         &Counter{},
	AnalyticsEnqueued:   &Counter{},
	AnalyticsDropped:    &Counter{},
	AnalyticsFlushed:    &Counter{},
	AnalyticsFailures:   &Counter{},
	ClicksBotFiltered:   &Counter{},
	URLsCreated:         &Counter{},
	URLsCreatedRejected: &Counter{},
}

// RateLimitRej returns (creating if needed) the rejection counter for a policy.
func RateLimitRej(policy string) *Counter {
	if v, ok := Registry.RateLimitRejected.Load(policy); ok {
		return v.(*Counter)
	}
	c := &Counter{}
	actual, _ := Registry.RateLimitRejected.LoadOrStore(policy, c)
	return actual.(*Counter)
}

// Snapshot renders the current metric values as a JSON-friendly map.
func Snapshot() map[string]any {
	rl := map[string]int64{}
	Registry.RateLimitRejected.Range(func(k, v any) bool {
		rl[k.(string)] = v.(*Counter).Value()
		return true
	})
	return map[string]any{
		"redirects_total":       Registry.RedirectTotal.Value(),
		"redirect_errors_total": Registry.RedirectErrors.Value(),
		"redirect_latency":      Registry.RedirectLatency.snapshot(),
		"cache_hits_total":      Registry.CacheHits.Value(),
		"cache_misses_total":    Registry.CacheMisses.Value(),
		"cache_negative_hits":   Registry.CacheNegativeHits.Value(),
		"cache_errors_total":    Registry.CacheErrors.Value(),
		"rate_limit_rejected":   rl,
		"urls_created_total":    Registry.URLsCreated.Value(),
		"urls_created_rejected": Registry.URLsCreatedRejected.Value(),
		"analytics_enqueued":    Registry.AnalyticsEnqueued.Value(),
		"analytics_dropped":     Registry.AnalyticsDropped.Value(),
		"analytics_flushed":     Registry.AnalyticsFlushed.Value(),
		"analytics_failures":    Registry.AnalyticsFailures.Value(),
		"clicks_bot_filtered":   Registry.ClicksBotFiltered.Value(),
	}
}
