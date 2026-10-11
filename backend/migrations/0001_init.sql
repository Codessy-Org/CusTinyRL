-- 0001_init.sql
-- Core schema for the URL shortener.

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE auth_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX auth_tokens_user_idx ON auth_tokens (user_id);
CREATE INDEX auth_tokens_expires_idx ON auth_tokens (expires_at);

CREATE TABLE short_urls (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    slug         TEXT NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    title        TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    click_count  BIGINT NOT NULL DEFAULT 0,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX short_urls_user_idx ON short_urls (user_id, id DESC);
CREATE INDEX short_urls_expires_idx ON short_urls (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX short_urls_active_slug_idx ON short_urls (slug) WHERE is_active;

CREATE TABLE click_events (
    id           BIGSERIAL PRIMARY KEY,
    short_url_id BIGINT NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
    clicked_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    referrer     TEXT,
    device       TEXT,
    country      TEXT
);

CREATE INDEX click_events_url_time_idx ON click_events (short_url_id, clicked_at DESC);
