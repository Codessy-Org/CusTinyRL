package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gauravkrrr/url-shortner/backend/internal/analytics"
	"github.com/gauravkrrr/url-shortner/backend/internal/apikey"
	"github.com/gauravkrrr/url-shortner/backend/internal/auth"
	"github.com/gauravkrrr/url-shortner/backend/internal/cache"
	"github.com/gauravkrrr/url-shortner/backend/internal/config"
	"github.com/gauravkrrr/url-shortner/backend/internal/db"
	"github.com/gauravkrrr/url-shortner/backend/internal/httpx"
	"github.com/gauravkrrr/url-shortner/backend/internal/metrics"
	"github.com/gauravkrrr/url-shortner/backend/internal/middleware"
	"github.com/gauravkrrr/url-shortner/backend/internal/preview"
	"github.com/gauravkrrr/url-shortner/backend/internal/ratelimit"
	"github.com/gauravkrrr/url-shortner/backend/internal/redirect"
	"github.com/gauravkrrr/url-shortner/backend/internal/url"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err.Error())
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "err", err.Error())
		os.Exit(1)
	}

	if cfg.AutoMigrate {
		ran, err := db.Migrate(ctx, pool)
		if err != nil {
			log.Error("migration failed", "err", err.Error())
			os.Exit(1)
		}
		if len(ran) > 0 {
			log.Info("applied migrations", "count", len(ran), "files", ran)
		}
	}

	rdb := cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB, cfg.CacheTTL, cfg.NegativeCacheTTL, cfg.CacheTimeout, log)
	limiter := ratelimit.New(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB, log)

	ipResolver, err := httpx.NewIPResolver(cfg.TrustedProxyCID)
	if err != nil {
		log.Error("invalid TRUSTED_PROXY_CIDRS", "err", err.Error())
		os.Exit(1)
	}

	urlRepo := url.NewRepository(pool)
	urlSvc := url.NewService(urlRepo, rdb, cfg.DefaultSlugLen, 5, cfg.BaseURL, log)
	if cfg.PreviewFetch {
		urlSvc.SetPreview(func(ctx context.Context, dest string) (string, string, error) {
			res, err := preview.Fetch(ctx, dest)
			return res.Title, res.FaviconURL, err
		})
	}
	urlHandler := url.NewHandler(urlSvc, log)

	ingestRepo := analytics.NewRepository(pool)
	ingester := analytics.NewIngester(ingestRepo, cfg.AnalyticsBufferSize, cfg.AnalyticsBatchSize, cfg.AnalyticsFlushInterval, log)
	ingester.SetHooks(
		func(n int) { metrics.Registry.AnalyticsFlushed.Add(int64(n)) },
		metrics.Registry.AnalyticsFailures.Inc,
	)
	go ingester.Run(ctx)

	analyticsHandler := analytics.NewHandler(analytics.NewQuerier(pool), urlSvc)

	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, cfg.AuthTokenTTL, log)
	authHandler := auth.NewHandler(authSvc, log)

	apiKeySvc := apikey.NewService(apikey.NewRepository(pool), log)
	authSvc.SetAPIKeyService(apiKeySvc)
	apiKeyHandler := apikey.NewHandler(apiKeySvc, log)

	redirectSvc := redirect.NewService(urlSvc, rdb, ingester, cfg.BotFilter, log)
	redirectHandler := redirect.NewHandler(redirectSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIP(ipResolver))
	r.Use(middleware.Recover(log))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logging(log))
	r.Use(middleware.Timeout(cfg.RequestTimeout))

	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/health/ready", func(w http.ResponseWriter, req *http.Request) {
		pctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pctx); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "dependency": "postgres"})
			return
		}
		if err := rdb.Ping(pctx); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "dependency": "redis"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Get("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, metrics.Snapshot())
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", auth.Routes(authHandler, authSvc.RequireAuth, limiter, cfg))
		r.With(authSvc.RequireAuth).Get("/me", authHandler.Me)
		r.Mount("/keys", apikey.Routes(apiKeyHandler, authSvc.RequireSession))
		r.Mount("/urls", url.Routes(urlHandler, authSvc.RequireAuth, authSvc.OptionalAuth, limiter, cfg, http.HandlerFunc(analyticsHandler.GetAnalytics)))
	})

	r.With(middleware.RateLimit(limiter, cfg.RateLimits["redirect"],
		func(req *http.Request) string {
			return "ip:" + middleware.GetClientIP(req.Context())
		},
	)).Get("/{slug}", redirectHandler.Go)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("api listening", "port", cfg.Port, "base_url", cfg.BaseURL, "anonymous_creation", cfg.AllowAnonymous)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err.Error())
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown error", "err", err.Error())
	}
	if !ingester.Wait(5 * time.Second) {
		log.Warn("analytics ingester did not drain before timeout; buffered events may be lost")
	}
	_ = rdb.Close()
	_ = limiter.Close()
	pool.Close()
	log.Info("shutdown complete")
}
