package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gauravkrrr/url-shortner/backend/internal/config"
	"github.com/gauravkrrr/url-shortner/backend/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error:", err)
		os.Exit(1)
	}
	defer pool.Close()

	ran, err := db.Migrate(ctx, pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration error:", err)
		os.Exit(1)
	}
	if len(ran) == 0 {
		fmt.Println("migrations: up to date")
		return
	}
	for _, name := range ran {
		fmt.Println("applied:", name)
	}
}
