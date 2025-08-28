package storage

import (
	"fmt"
	"rsshub/internal/platform/logger"
)

// RunMigrations запускает все миграции базы данных
func (db *DB) RunMigrations() error {
	logger.Info("Running database migrations...")

	// Создаем расширение для UUID
	if err := db.createUUIDExtension(); err != nil {
		return fmt.Errorf("failed to create UUID extension: %w", err)
	}

	// Создаем таблицу feeds
	if err := db.createFeedsTable(); err != nil {
		return fmt.Errorf("failed to create feeds table: %w", err)
	}

	// Создаем таблицу articles
	if err := db.createArticlesTable(); err != nil {
		return fmt.Errorf("failed to create articles table: %w", err)
	}

	logger.Success("Database migrations completed successfully")
	return nil
}

// createUUIDExtension создает расширение для работы с UUID
func (db *DB) createUUIDExtension() error {
	query := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

	_, err := db.Exec(query)
	return err
}

// createFeedsTable создает таблицу feeds
func (db *DB) createFeedsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS feeds (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			name TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL
		);

		-- Создаем индексы если они не существуют
		CREATE INDEX IF NOT EXISTS idx_feeds_name ON feeds(name);
		CREATE INDEX IF NOT EXISTS idx_feeds_updated_at ON feeds(updated_at);
	`

	_, err := db.Exec(query)
	return err
}

// createArticlesTable создает таблицу articles
func (db *DB) createArticlesTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS articles (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			title TEXT NOT NULL,
			link TEXT NOT NULL,
			published_at TIMESTAMP,
			description TEXT,
			feed_id UUID NOT NULL,
			
			-- Ограничения
			FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
			UNIQUE(link)
		);

		-- Создаем индексы если они не существуют
		CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
		CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at DESC);
		CREATE INDEX IF NOT EXISTS idx_articles_feed_published ON articles(feed_id, published_at DESC);
	`

	_, err := db.Exec(query)
	return err
}

// table aggregator settings
func (db *DB) createAggregatorTable() error {
	query := `
		CREATE TABLE aggregator (
    		id SERIAL PRIMARY KEY,
    		key TEXT UNIQUE NOT NULL,
    		value TEXT NOT NULL);

		-- Индекс
		CREATE INDEX idx_aggregator_id ON aggregator(id);
	`

	_, err := db.Exec(query)
	return err
}
