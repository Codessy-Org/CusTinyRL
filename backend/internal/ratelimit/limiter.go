// Package ratelimit implements Redis-backed fixed-window rate limiting that
// is shared across API instances. Redis outages fail open (availability
// first); the failure is logged and counted.
package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gauravkrrr/url-shortner/backend/internal/config"
)

// allowScript performs an atomic INCR + PEXPIRE fixed window.
// Returns {allowed(0/1), ttl_ms, remaining}.
var allowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local current = redis.call('INCR', key)
if current == 1 then
  redis.call('PEXPIRE', key, window_ms)
end
local ttl = redis.call('PTTL', key)
if ttl < 0 then
  redis.call('PEXPIRE', key, window_ms)
  ttl = window_ms
end
if current > limit then
  return {0, ttl, 0}
end
return {1, ttl, limit - current}
`)

type Result struct {
	Allowed    bool
	RetryAfter time.Duration
	Remaining  int
}

type Limiter struct {
	client *redis.Client
	log    *slog.Logger
}

func New(addr, password string, db int, log *slog.Logger) *Limiter {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	return &Limiter{client: client, log: log}
}

func (l *Limiter) Close() error { return l.client.Close() }

// Allow consumes one unit from the policy window for subject.
// subject should be "user:<id>" or "ip:<addr>".
func (l *Limiter) Allow(ctx context.Context, policy config.RateLimitPolicy, subject string) (Result, error) {
	key := "rl:" + policy.Name + ":" + subject
	res, err := allowScript.Run(ctx, l.client, []string{key},
		policy.Limit, policy.Window.Milliseconds(), time.Now().UnixMilli()).Int64Slice()
	if err != nil {
		// Fail open when Redis is unavailable.
		if l.log != nil {
			l.log.Warn("rate limit check failed, allowing request", "policy", policy.Name, "err", err.Error())
		}
		return Result{Allowed: true, Remaining: policy.Limit}, nil
	}
	if len(res) != 3 {
		return Result{Allowed: true, Remaining: policy.Limit}, nil
	}
	allowed := res[0] == 1
	out := Result{Allowed: allowed, Remaining: int(res[2])}
	if !allowed {
		ttl := time.Duration(res[1]) * time.Millisecond
		if ttl <= 0 {
			ttl = policy.Window
		}
		out.RetryAfter = ttl.Round(time.Second)
		if out.RetryAfter < time.Second {
			out.RetryAfter = time.Second
		}
	}
	return out, nil
}
