package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type RateLimitPolicy struct {
	Name   string
	Limit  int
	Window time.Duration
}

type Config struct {
	Port     string
	LogLevel string

	DatabaseURL string
	RedisAddr   string
	RedisPass   string
	RedisDB     int

	BaseURL         string
	AllowedOrigins  []string
	TrustedProxyCID []string

	AllowAnonymous bool
	AuthTokenTTL   time.Duration

	DefaultSlugLen int
	MaxSlugLen     int

	CacheTTL         time.Duration
	NegativeCacheTTL time.Duration
	CacheTimeout     time.Duration

	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration

	AnalyticsBufferSize    int
	AnalyticsBatchSize     int
	AnalyticsFlushInterval time.Duration

	PreviewFetch bool
	BotFilter    bool

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string

	AutoMigrate bool

	RateLimits map[string]RateLimitPolicy
}

func Load() (*Config, error) {
	loadDotEnv()
	c := &Config{
		Port:                   getEnv("PORT", "8080"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:              os.Getenv("REDIS_PASSWORD"),
		RedisDB:                getEnvInt("REDIS_DB", 0),
		BaseURL:                strings.TrimRight(getEnv("BASE_URL", "http://localhost:8080"), "/"),
		AllowedOrigins:         splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:3000")),
		TrustedProxyCID:        splitCSV(getEnv("TRUSTED_PROXY_CIDRS", "")),
		AllowAnonymous:         getEnvBool("ALLOW_ANONYMOUS", true),
		AuthTokenTTL:           getEnvDuration("AUTH_TOKEN_TTL", 30*24*time.Hour),
		DefaultSlugLen:         getEnvInt("SLUG_LENGTH", 10),
		MaxSlugLen:             64,
		CacheTTL:               getEnvDuration("CACHE_TTL", 10*time.Minute),
		NegativeCacheTTL:       getEnvDuration("NEGATIVE_CACHE_TTL", 30*time.Second),
		CacheTimeout:           getEnvDuration("CACHE_TIMEOUT", 150*time.Millisecond),
		RequestTimeout:         getEnvDuration("REQUEST_TIMEOUT", 10*time.Second),
		ShutdownTimeout:        getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		AnalyticsBufferSize:    getEnvInt("ANALYTICS_BUFFER_SIZE", 1024),
		AnalyticsBatchSize:     getEnvInt("ANALYTICS_BATCH_SIZE", 100),
		AnalyticsFlushInterval: getEnvDuration("ANALYTICS_FLUSH_INTERVAL", 1*time.Second),
		PreviewFetch:           getEnvBool("PREVIEW_FETCH", true),
		BotFilter:              getEnvBool("BOT_FILTER", true),
		GoogleClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:      getEnv("GOOGLE_REDIRECT_URL", ""),
		FrontendURL:            strings.TrimRight(getEnv("FRONTEND_URL", "http://localhost:3000"), "/"),
		AutoMigrate:            getEnvBool("AUTO_MIGRATE", false),
	}
	if c.GoogleRedirectURL == "" {
		c.GoogleRedirectURL = c.BaseURL + "/api/v1/auth/google/callback"
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.DefaultSlugLen < 4 || c.DefaultSlugLen > c.MaxSlugLen {
		return nil, fmt.Errorf("SLUG_LENGTH must be between 4 and %d", c.MaxSlugLen)
	}

	c.RateLimits = map[string]RateLimitPolicy{
		"url_create":  {Name: "url_create", Limit: getEnvInt("RL_URL_CREATE_LIMIT", 10), Window: getEnvDuration("RL_URL_CREATE_WINDOW", time.Minute)},
		"anon_create": {Name: "anon_create", Limit: getEnvInt("RL_ANON_CREATE_LIMIT", 5), Window: getEnvDuration("RL_ANON_CREATE_WINDOW", time.Minute)},
		"auth":        {Name: "auth", Limit: getEnvInt("RL_AUTH_LIMIT", 10), Window: getEnvDuration("RL_AUTH_WINDOW", time.Minute)},
		"analytics":   {Name: "analytics", Limit: getEnvInt("RL_ANALYTICS_LIMIT", 60), Window: getEnvDuration("RL_ANALYTICS_WINDOW", time.Minute)},
		"redirect":    {Name: "redirect", Limit: getEnvInt("RL_REDIRECT_LIMIT", 300), Window: getEnvDuration("RL_REDIRECT_WINDOW", time.Minute)},
	}
	return c, nil
}

// loadDotEnv reads a .env file from the working directory, if present.
// Existing environment variables always win, so `FOO=1 go run ...` and
// containerized env injection keep their expected precedence. Values may be
// quoted; blank lines and lines starting with # are ignored.
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, " \t") {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, parseDotEnvValue(strings.TrimSpace(value)))
	}
}

// parseDotEnvValue strips surrounding quotes and inline `#` comments.
// Inside quotes, content is taken verbatim (so fragments/`#` survive).
func parseDotEnvValue(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
		quote := value[0]
		if end := strings.LastIndexByte(value, quote); end > 0 {
			return value[1:end]
		}
		return value[1:]
	}
	// Unquoted: an inline comment starts at `#` preceded by whitespace (or at
	// the start of the value).
	if idx := strings.IndexByte(value, '#'); idx >= 0 {
		if idx == 0 {
			return ""
		}
		if c := value[idx-1]; c == ' ' || c == '\t' {
			value = strings.TrimSpace(value[:idx])
		}
	}
	return value
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
