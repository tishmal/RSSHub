package main

import (
	"os"

	"rsshub/internal/adapter/cli"
	httpfetcher "rsshub/internal/adapter/fetcher/http"
	"rsshub/internal/adapter/storage"
	"rsshub/internal/platform/config"
	"rsshub/internal/platform/logger"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Connect to DB
	db, err := storage.New(cfg.Database.GetDSN())
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("Error closing database: %v", err)
		}
	}()

	// 3. Run migrations
	if err := db.RunMigrations(); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}

	parser := httpfetcher.NewParser()

	// 4. Build CLI (composition root: inject repository + config)
	cliApp := cli.New(db, parser, cfg)

	// 5. Run CLI
	if err := cliApp.Run(os.Args); err != nil {
		logger.Error("Command failed: %v", err)
		os.Exit(1)
	}
}
