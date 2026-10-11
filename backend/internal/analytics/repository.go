package analytics

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Point struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type LabelCount struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type Summary struct {
	TotalClicks  int64        `json:"total_clicks"`
	Clicks24h    int64        `json:"clicks_24h"`
	Series       []Point      `json:"series"`
	TopReferrers []LabelCount `json:"top_referrers"`
	Devices      []LabelCount `json:"devices"`
}

// Querier aggregates click analytics for a single short URL.
type Querier struct {
	pool *pgxpool.Pool
}

func NewQuerier(pool *pgxpool.Pool) *Querier { return &Querier{pool: pool} }

func (q *Querier) Summary(ctx context.Context, shortURLID int64, days int) (*Summary, error) {
	if days < 1 || days > 90 {
		days = 30
	}
	s := &Summary{Series: []Point{}, TopReferrers: []LabelCount{}, Devices: []LabelCount{}}

	if err := q.pool.QueryRow(ctx,
		`SELECT click_count FROM short_urls WHERE id = $1`, shortURLID).Scan(&s.TotalClicks); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	since := time.Now().Add(-24 * time.Hour)
	if err := q.pool.QueryRow(ctx,
		`SELECT count(*) FROM click_events WHERE short_url_id = $1 AND clicked_at >= $2`,
		shortURLID, since).Scan(&s.Clicks24h); err != nil {
		return nil, err
	}

	rows, err := q.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', clicked_at), 'YYYY-MM-DD') AS d, count(*)
		FROM click_events
		WHERE short_url_id = $1 AND clicked_at >= now() - make_interval(days => $2::int)
		GROUP BY 1 ORDER BY 1`, shortURLID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			return nil, err
		}
		s.Series = append(s.Series, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows2, err := q.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(referrer, ''), 'direct'), count(*)
		FROM click_events
		WHERE short_url_id = $1 AND clicked_at >= now() - make_interval(days => $2::int)
		GROUP BY 1 ORDER BY 2 DESC, 1 LIMIT 5`, shortURLID, days)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var lc LabelCount
		if err := rows2.Scan(&lc.Label, &lc.Count); err != nil {
			return nil, err
		}
		s.TopReferrers = append(s.TopReferrers, lc)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	rows3, err := q.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(device, ''), 'unknown'), count(*)
		FROM click_events
		WHERE short_url_id = $1 AND clicked_at >= now() - make_interval(days => $2::int)
		GROUP BY 1 ORDER BY 2 DESC, 1 LIMIT 5`, shortURLID, days)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var lc LabelCount
		if err := rows3.Scan(&lc.Label, &lc.Count); err != nil {
			return nil, err
		}
		s.Devices = append(s.Devices, lc)
	}
	if err := rows3.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

// ErrNotFound when the short URL does not exist.
var ErrNotFound = errors.New("not found")
