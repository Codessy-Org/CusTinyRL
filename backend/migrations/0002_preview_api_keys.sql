-- 0002_preview_api_keys.sql
-- Link-preview metadata and scoped API keys.

ALTER TABLE short_urls ADD COLUMN favicon_url TEXT;

CREATE TABLE api_keys (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    key_prefix    TEXT NOT NULL,
    key_hash      BYTEA NOT NULL UNIQUE,
    scopes        TEXT[] NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ
);

CREATE INDEX api_keys_user_idx ON api_keys (user_id, id DESC);
