# cut.sh — Production-Ready URL Shortener

A production-oriented URL shortener with custom slugs, high-performance cached
redirects, distributed rate limiting, click analytics, QR codes, scoped API
keys, and a CLI.

- **Frontend:** Next.js 16 (App Router) + TypeScript + Tailwind CSS 4 — animated landing page + Vercel-inspired dashboard
- **Backend:** Go + Chi router, `pgxpool`, structured JSON logging, graceful shutdown
- **Data:** PostgreSQL (source of truth), Redis (resolution cache + rate limiting)

```mermaid
flowchart LR
    B[Browser / Client] -->|HTTPS| W[Next.js dashboard :3000]
    B -->|GET /{slug}| A[Go API :8080]
    W -->|REST /api/v1/*| A
    subgraph Go API
        M[Chi middleware\nrequest-id · CORS · timeout · rate limit]
        H[Handlers]
        S[Services]
        M --> H --> S
    end
    S -->|resolution cache| R[(Redis)]
    S -->|source of truth| P[(PostgreSQL)]
    A -->|async batched insert| P
    R -.->|miss / invalidation| S
```

## Repository layout

```
.
├── frontend/                 # Next.js: landing page + dashboard (Vercel-inspired UI)
│   ├── app/                  # App Router pages, layout, design tokens
│   ├── components/           # CreateForm, UrlTable, modals, theme, toasts
│   ├── lib/                  # API client, types, formatting helpers
│   └── __tests__/            # Vitest + Testing Library component tests
├── backend/                  # Go module
│   ├── cmd/api/              # HTTP server entrypoint
│   ├── cmd/cutsh/            # CLI client (login, add, list, stats, delete)
│   ├── cmd/migrate/          # Migration runner
│   ├── internal/
│   │   ├── config/           # Environment configuration
│   │   ├── url/              # Slug generation, validation, CRUD, previews
│   │   ├── redirect/         # Public /{slug} resolution
│   │   ├── cache/            # Redis cache (+ stampede protection)
│   │   ├── ratelimit/        # Redis fixed-window limiter (Lua)
│   │   ├── analytics/        # Async click ingestion + aggregation + bot filter
│   │   ├── auth/             # Email/password + opaque bearer tokens + API keys
│   │   ├── apikey/           # Scoped API keys (ck_*, create/read/manage)
│   │   ├── preview/          # SSRF-guarded title/favicon fetch
│   │   ├── middleware/       # request-id, logging, CORS, rate limits
│   │   ├── httpx/            # JSON responses, trusted-proxy client IP
│   │   ├── identity/         # Authenticated user + scopes in request context
│   │   ├── metrics/          # In-process counters/histograms (/metrics)
│   │   └── db/               # pgxpool + migrator
│   └── migrations/           # Ordered SQL migrations (embedded)
├── docker-compose.yml        # postgres + redis + api + web
└── .env.example
```

## Local setup

### Option A — Docker Compose (recommended)

```bash
cp .env.example .env            # adjust secrets for anything non-local
docker compose up --build
```

- Landing page: http://localhost:3000 (animated product page)
- Dashboard: http://localhost:3000/dashboard (app — sign in to manage links)
- API: http://localhost:8080 (health: `/health/live`, `/health/ready`, `/metrics`)
- Migrations run automatically before the API starts (`migrate` service).

### Option B — run services natively (apps outside containers)

Run only the data stores in Docker; run the API and dashboard on the host.

```bash
# 1. Data stores
docker compose up -d db redis

# 2. Backend — .env is loaded automatically from this directory
cd backend
cp .env.example .env
go run ./cmd/api

# 3. Frontend (new terminal)
cd frontend
cp .env.example .env.local
npm ci
npm run dev
```

Notes:

- `backend/.env` and `frontend/.env.local` are read automatically — no `export`,
  no shell sourcing. Real environment variables always override file values.
- Set `AUTO_MIGRATE=true` in `backend/.env` to apply migrations at startup
  (the default in `.env.example` is `false`; run `go run ./cmd/migrate` for an
  explicit, production-style migration step instead).
- If port 8080 is already in use, set `PORT`/`BASE_URL` in `backend/.env` and
  point `NEXT_PUBLIC_API_URL` at the same origin.
- Never paste shell snippets that mix commands and `# comments` — some shells
  treat the comment text as extra arguments.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | *(required)* | Postgres DSN |
| `PORT` | `8080` | API listen port |
| `BASE_URL` | `http://localhost:8080` | Origin used in returned short URLs |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` / `REDIS_DB` | empty / `0` | Redis auth / logical DB |
| `CACHE_TTL` | `10m` | Positive cache TTL |
| `NEGATIVE_CACHE_TTL` | `30s` | Missing-slug cache TTL |
| `CACHE_TIMEOUT` | `150ms` | Bound on Redis reads |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | CORS allowlist (comma separated) |
| `TRUSTED_PROXY_CIDRS` | empty | Proxies allowed to set `X-Forwarded-For` |
| `ALLOW_ANONYMOUS` | `true` | Permit unauthenticated creation (IP-limited) |
| `AUTH_TOKEN_TTL` | `720h` | Bearer token lifetime |
| `SLUG_LENGTH` | `10` | Generated slug length (4–64) |
| `RL_*_LIMIT` / `RL_*_WINDOW` | see `backend/.env.example` | Per-policy rate limits |
| `ANALYTICS_BUFFER_SIZE` | `1024` | In-memory event buffer |
| `ANALYTICS_BATCH_SIZE` / `ANALYTICS_FLUSH_INTERVAL` | `100` / `1s` | Batch insert tuning |
| `BOT_FILTER` | `true` | Skip analytics for known bot/crawler user agents |
| `PREVIEW_FETCH` | `true` | Fetch destination title + favicon on create (SSRF-guarded) |
| `AUTO_MIGRATE` | `false` | Apply migrations at startup |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Frontend → API origin (build-time) |

Copy `.env.example`, `backend/.env.example`, and `frontend/.env.example` — they
contain placeholders only.

## Database migrations

Migrations are embedded, ordered `.sql` files guarded by a Postgres advisory
lock and recorded in `schema_migrations`.

```bash
# standalone
cd backend && go run ./cmd/migrate

# or via compose (runs automatically)
docker compose run --rm migrate

# or set AUTO_MIGRATE=true on the API (not recommended for production)
```

To add a migration: create `backend/migrations/0002_<name>.sql` (never edit an
applied file).

## API reference

All API endpoints return JSON. Errors use a consistent shape:

```json
{ "error": { "code": "slug_taken", "message": "that slug is already taken" } }
```

Authentication: `Authorization: Bearer <token>` (issued by register/login).

### Auth

```bash
# Register → returns a bearer token (shown once; stored as SHA-256 server-side)
curl -s -X POST localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}'

# Login
curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}'

curl -s localhost:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
curl -s -X POST localhost:8080/api/v1/auth/logout -H "Authorization: Bearer $TOKEN"
```

### Create URLs

```bash
# Custom slug (409 if taken; input is never silently altered)
curl -s -X POST localhost:8080/api/v1/urls \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"original_url":"https://docs.google.com/document/d/example-document-id",
       "custom_slug":"gauravresumev1"}'

# Automatic Base62 slug with optional prefix → resume-a7Kp9xQ2Lm
curl -s -X POST localhost:8080/api/v1/urls \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com","prefix":"resume","expires_at":"2027-01-01T00:00:00Z"}'
```

Anonymous creation works when `ALLOW_ANONYMOUS=true` (strict per-IP limits).
Anonymous creates get an **automatic slug only** — `custom_slug` and `prefix`
require a signed-in user (API key or session) and return `403 auth_required`
otherwise.

### Manage URLs

```bash
curl -s "localhost:8080/api/v1/urls?limit=20&cursor=42&q=resume" -H "Authorization: Bearer $TOKEN"
curl -s localhost:8080/api/v1/urls/12 -H "Authorization: Bearer $TOKEN"
curl -s -X PATCH localhost:8080/api/v1/urls/12 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.org","is_active":false}'
curl -s -X DELETE localhost:8080/api/v1/urls/12 -H "Authorization: Bearer $TOKEN"
```

### Redirects, analytics, and previews

```bash
curl -i localhost:8080/gauravresumev1          # 302 → Location
curl -s localhost:8080/api/v1/urls/12/analytics?days=30 -H "Authorization: Bearer $TOKEN"
```

Creates fetch the destination's title and favicon (bounded, SSRF-guarded:
loopback/private/metadata addresses are blocked; disable with
`PREVIEW_FETCH=false`). Redirect hits from known bots are excluded from click
analytics when `BOT_FILTER=true`.

### API keys (scoped, for automation)

```bash
# Create a key (raw key returned once; hashed server-side). Scopes: create, read, manage.
curl -s -X POST localhost:8080/api/v1/keys -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"ci-pipeline","scopes":["create"]}'

curl -s localhost:8080/api/v1/keys -H "Authorization: Bearer $TOKEN"
curl -s -X DELETE localhost:8080/api/v1/keys/1 -H "Authorization: Bearer $TOKEN"

# Use a key exactly like a session token (missing scope → 403; revoked/expired → 401)
curl -s -X POST localhost:8080/api/v1/urls \
  -H "Authorization: Bearer ck_..." -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com","prefix":"ci"}'
```

Keys require a session token to manage (keys cannot manage keys). Anonymous
creation is never a fallback: a revoked or expired key gets `401`, not silent
anonymous access.

### CLI

```bash
cd backend && go build -o cutsh ./cmd/cutsh

cutsh login --email you@example.com          # token stored in ~/.config/cutsh/config.json
cutsh add https://example.com/résumé --title "My resume"
cutsh add https://example.com --slug my-exact-slug --prefix resume
cutsh list --limit 20 --q resume
cutsh stats <id> --days 30
cutsh delete <id>
cutsh whoami && cutsh logout
```

`CUTSH_API_URL` overrides the API origin; `CUTSH_TOKEN` overrides the stored
token (useful for CI with an API key).

### Status codes

| Code | Meaning |
|---|---|
| `200` / `201` / `204` | Success |
| `401` | Missing/invalid token |
| `403` | `auth_required` — custom slug/prefix needs a signed-in user |
| `404` | Unknown id or slug |
| `409` | `slug_taken` — custom slug already exists |
| `410` | Link disabled or expired |
| `422` | `validation_error` — bad URL/slug/expiry |
| `429` | `rate_limited` + `Retry-After` header |

## Slug generation strategy

- **Custom slugs:** canonicalized to **lowercase**, charset
  `^[a-z0-9][a-z0-9_-]{0,63}$`, checked against a reserved list
  (`api`, `login`, `dashboard`, `health`, `_next`, …). Uniqueness is enforced
  by the Postgres unique constraint — never by a preceding SELECT — so
  concurrent creates cannot race. Collisions return **409**.
- **Generated slugs:** `crypto/rand` bytes mapped to a URL-safe **Base62**
  alphabet with rejection sampling (unbiased), default 10 characters.
  The DB unique constraint is the arbiter: inserts use
  `ON CONFLICT DO NOTHING` + bounded retries (5) — hashing or encoding alone
  is never trusted for uniqueness.
- **Prefixes** apply only to generated slugs: `resume` + `a7Kp9xQ2Lm` →
  `resume-a7Kp9xQ2Lm` (lowercase, ≤ 32 chars, total ≤ 64). Prefixes never
  modify user-supplied custom slugs.
- Redirect lookup is **case-sensitive** (generated slugs are mixed-case).

## Redis caching strategy

Key format `url:{slug}` → JSON record `{id, slug, original_url, is_active, expires_at}`.

1. `GET url:{slug}`; hit → validate active/expiry from the record (an expired
   or disabled cached record can never redirect) → **302**.
2. Miss → singleflight-coalesced Postgres lookup (stampede protection) →
   populate cache with TTL `CACHE_TTL` (10m) or `expires_at`, whichever is sooner.
3. Missing slug → negative-cached for `NEGATIVE_CACHE_TTL` (30s) to shield
   Postgres from probe traffic.
4. Create/update/delete **invalidate** `url:{slug}` immediately, so stale
   destinations are never served after a write.
5. Redis is a cache only — never the source of truth. On Redis errors requests
   degrade to Postgres with a warning (availability first) and timeouts are
   bounded (`CACHE_TIMEOUT`).

## Rate limiting

Redis fixed-window via an atomic Lua script (`INCR` + `PEXPIRE`), shared
across all API instances. Exceeding a policy returns `429` with `Retry-After`.

| Policy | Default | Subject |
|---|---|---|
| `url_create` | 10 / min | user id (IP for anonymous) |
| `anon_create` | 5 / min | IP |
| `auth` (register/login) | 10 / min | IP |
| `analytics` | 60 / min | user id |
| `redirect` | 300 / min | IP |

- Client IPs are resolved through `TRUSTED_PROXY_CIDRS`; `X-Forwarded-For` is
  ignored unless the direct peer is a trusted proxy (no spoofing).
- If Redis is down, limits **fail open** (availability first) and the failure
  is logged + counted.
- Application-level limits complement — never replace — edge protection
  (Cloudflare rate limiting / WAF / bot management).

## Redirect request flow

1. Validate/canonicalize slug (`[A-Za-z0-9_-]{1,64}`) → `404` if malformed
2. Per-IP redirect rate limit → `429`
3. Redis lookup → Postgres fallback → `404` unknown, `410` disabled/expired
4. Enqueue click event (host-only referrer, coarse device category, **no IP**;
   known bots are skipped when `BOT_FILTER=true`)
5. `302 Found` with `Cache-Control: no-store`

## Testing

```bash
# Backend unit tests (no services required)
cd backend && go test ./...

# Backend integration tests (Postgres + Redis required)
cd backend
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/urlshort?sslmode=disable" \
TEST_REDIS_ADDR="localhost:6379" \
go test ./... -count=1

# Race detector
go test -race ./...

# Benchmarks (live services; numbers are printed, never hardcoded)
TEST_DATABASE_URL=... go test ./internal/integration -bench . -benchtime 2s

# Frontend unit/component tests
cd frontend && npm test

# Lint + production builds
cd frontend && npm run lint && npm run build
cd backend && go vet ./... && go build ./...
```

Integration tests are skipped automatically when `TEST_DATABASE_URL` is unset.

## Deployment

- **TLS / reverse proxy:** terminate TLS at a reverse proxy (Caddy, nginx,
  traefik, or a managed load balancer) and forward to the API (`:8080`) and
  web (`:3000`) containers. Set `TRUSTED_PROXY_CIDRS` to the proxy's network
  so client IPs resolve correctly. Set `BASE_URL` to your public API origin.
- **Compose:** `docker compose up -d --build` after creating `.env`.
  Postgres and Redis are **not** published to the host; only the API (`8080`)
  and web (`3000`) ports are exposed. Postgres data lives in the `pgdata`
  named volume. Health checks gate startup (`db`/`redis` healthy → `migrate`
  completes → `api` starts).
- **Secrets** stay in environment files excluded from source control.
- **Scaling:** the API is stateless — run multiple replicas behind the proxy;
  rate limiting and caching are already shared through Redis. Set
  `TRUSTED_PROXY_CIDRS` correctly before scaling horizontally.

## Observability

- Structured JSON logs with `request_id`, method, path, status, latency, IP.
- `GET /health/live` — process liveness; `GET /health/ready` — Postgres + Redis.
- `GET /metrics` — JSON counters: redirect totals/latency histogram, cache
  hit/miss/negative/error counts, rate-limit rejections per policy, analytics
  enqueue/flush/drop/failure counts.

## Known limitations & next steps

1. **Analytics durability:** click events buffer in an in-process channel and
   batch-insert every ~1s. Events buffered at crash time are **lost** — this is
   not a durable pipeline. For durability, publish to Redis Streams/Kafka and
   consume with retries.
2. **No unique-visitor tracking:** only aggregate clicks are stored; IP
   addresses are never persisted. Add privacy-reviewed, salted, rolling-window
   IP hashing only if unique visitors become a requirement.
3. **Auth is deliberately minimal** (email + password, opaque tokens). Add
   email verification, token rotation, refresh tokens, and 2FA as needed; OTP
   delivery would need an SMTP provider.
4. **Geo/country** fields exist in the schema but stay empty — populate only
   from a trusted geo-IP source.
5. **Fixed-window rate limiting** allows brief bursts at window edges;
   switch to a sliding-window Lua script if that matters.
6. **Link previews are best-effort regex parsing** — good enough for titles
   and favicons, not a full browser. Add oEmbed/OpenGraph parsing with the
   same SSRF guard if richer metadata is needed.
7. **Domain allowlisting for redirects** (anti-phishing) and a reporting
   workflow are future work; `is_active` + 410 are the current controls.
