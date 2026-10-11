package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gauravkrrr/url-shortner/backend/migrations"
)

// Migrate applies ordered .sql migrations in a transaction per file,
// guarded by a Postgres advisory lock so concurrent migrators are safe.
func Migrate(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	return MigrateFS(ctx, pool, migrations.FS, ".")
}

// MigrateFS applies migrations from an arbitrary fs rooted at dir.
func MigrateFS(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, dir string) ([]string, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	const lockKey int64 = 72727401
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		return nil, fmt.Errorf("advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockKey)
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	appliedRows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	applied := map[string]bool{}
	for appliedRows.Next() {
		var v string
		if err := appliedRows.Scan(&v); err != nil {
			appliedRows.Close()
			return nil, err
		}
		applied[v] = true
	}
	appliedRows.Close()

	var ran []string
	for _, name := range names {
		if applied[name] {
			continue
		}
		path := name
		if dir != "." && dir != "" {
			path = dir + "/" + name
		}
		body, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return nil, fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		ran = append(ran, name)
	}
	return ran, nil
}
