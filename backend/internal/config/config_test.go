package config

import (
	"testing"
	"time"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without DATABASE_URL")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultSlugLen != 10 {
		t.Fatalf("DefaultSlugLen = %d", cfg.DefaultSlugLen)
	}
	if cfg.MaxSlugLen != 64 {
		t.Fatalf("MaxSlugLen = %d", cfg.MaxSlugLen)
	}
	if !cfg.AllowAnonymous {
		t.Fatal("AllowAnonymous should default true")
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Fatalf("CacheTTL = %v", cfg.CacheTTL)
	}
	p := cfg.RateLimits["url_create"]
	if p.Limit != 10 || p.Window != time.Minute {
		t.Fatalf("url_create policy = %+v", p)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("SLUG_LENGTH", "12")
	t.Setenv("ALLOW_ANONYMOUS", "false")
	t.Setenv("RL_URL_CREATE_LIMIT", "20")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultSlugLen != 12 || cfg.AllowAnonymous {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
	if cfg.RateLimits["url_create"].Limit != 20 {
		t.Fatal("rate limit override not applied")
	}
}

func TestLoadRejectsBadSlugLength(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("SLUG_LENGTH", "3")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for SLUG_LENGTH=3")
	}
}
