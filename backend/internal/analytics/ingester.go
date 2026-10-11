// Package analytics ingests click events asynchronously. Events travel
// through an in-memory buffered channel and are written in batches; events
// buffered at process crash time can be lost. When durable analytics are
// required, back this package with a durable queue (e.g. Redis Streams with
// AOF, Kafka, or Kinesis) instead of a process-local channel.
package analytics

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is a single redirect click observation.
// Country is reserved for a trusted geo-IP source and stays empty for now;
// raw IP addresses are deliberately never stored.
type Event struct {
	ShortURLID int64
	ClickedAt  time.Time
	Referrer   string
	Device     string
	Country    string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// InsertBatch writes a batch of events and increments the denormalized click
// counter on short_urls in one transaction.
func (r *Repository) InsertBatch(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, e := range events {
		if _, err := tx.Exec(ctx, `
			INSERT INTO click_events (short_url_id, clicked_at, referrer, device, country)
			VALUES ($1, COALESCE($2, now()), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''))`,
			e.ShortURLID, e.ClickedAt, e.Referrer, e.Device, e.Country); err != nil {
			return err
		}
	}
	counts := map[int64]int64{}
	for _, e := range events {
		counts[e.ShortURLID]++
	}
	for id, n := range counts {
		if _, err := tx.Exec(ctx,
			`UPDATE short_urls SET click_count = click_count + $2 WHERE id = $1`, id, n); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// BatchStore abstracts batch persistence for tests.
type BatchStore interface {
	InsertBatch(ctx context.Context, events []Event) error
}

// Ingester buffers events from the hot redirect path and flushes batches.
type Ingester struct {
	repo        BatchStore
	ch          chan Event
	batch       int
	every       time.Duration
	log         *slog.Logger
	done        chan struct{}
	stopped     atomic.Bool
	onFlushed   func(n int)
	onFlushFail func()
}

func NewIngester(repo BatchStore, bufferSize, batchSize int, every time.Duration, log *slog.Logger) *Ingester {
	if batchSize < 1 {
		batchSize = 100
	}
	if bufferSize < batchSize {
		bufferSize = batchSize
	}
	return &Ingester{
		repo:  repo,
		ch:    make(chan Event, bufferSize),
		batch: batchSize,
		every: every,
		log:   log,
		done:  make(chan struct{}),
	}
}

// SetHooks wires metrics callbacks without importing the metrics package.
func (i *Ingester) SetHooks(onFlushed func(n int), onFlushFail func()) {
	i.onFlushed = onFlushed
	i.onFlushFail = onFlushFail
}

// Record enqueues an event without blocking the redirect path. When the
// buffer is full the event is dropped and counted via onDropped. It reports
// whether the event was accepted.
func (i *Ingester) Record(ev Event, onDropped func()) bool {
	if i.stopped.Load() {
		if onDropped != nil {
			onDropped()
		}
		return false
	}
	select {
	case i.ch <- ev:
		return true
	default:
		if onDropped != nil {
			onDropped()
		}
		return false
	}
}

// Run flushes until ctx is canceled, then drains remaining events.
func (i *Ingester) Run(ctx context.Context) {
	ticker := time.NewTicker(i.every)
	defer ticker.Stop()
	pending := make([]Event, 0, i.batch)

	flush := func() {
		if len(pending) == 0 {
			return
		}
		batch := pending
		pending = make([]Event, 0, i.batch)
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		err := i.repo.InsertBatch(wctx, batch)
		cancel()
		if err != nil {
			if i.log != nil {
				i.log.Error("analytics batch insert failed", "count", len(batch), "err", err.Error())
			}
			if i.onFlushFail != nil {
				i.onFlushFail()
			}
			return
		}
		if i.onFlushed != nil {
			i.onFlushed(len(batch))
		}
	}

	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case ev := <-i.ch:
					pending = append(pending, ev)
					if len(pending) >= i.batch {
						flush()
					}
				default:
					flush()
					i.stopped.Store(true)
					close(i.done)
					return
				}
			}
		case ev := <-i.ch:
			pending = append(pending, ev)
			if len(pending) >= i.batch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Wait blocks until Run has finished its final drain (for graceful shutdown).
func (i *Ingester) Wait(timeout time.Duration) bool {
	select {
	case <-i.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// SanitizeReferrer keeps only the scheme-stripped host, truncated.
func SanitizeReferrer(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	ref = strings.TrimPrefix(ref, "https://")
	ref = strings.TrimPrefix(ref, "http://")
	if i := strings.IndexAny(ref, "/?#"); i >= 0 {
		ref = ref[:i]
	}
	ref = strings.ToLower(ref)
	if len(ref) > 100 {
		ref = ref[:100]
	}
	return ref
}
