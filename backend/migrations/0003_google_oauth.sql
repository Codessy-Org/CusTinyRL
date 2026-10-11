-- Google OAuth: passwordless users have password_hash NULL; google_sub
-- links the Google identity (unique) and stores display fields.
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE users
    ADD COLUMN google_sub TEXT UNIQUE,
    ADD COLUMN name       TEXT,
    ADD COLUMN avatar_url TEXT;
