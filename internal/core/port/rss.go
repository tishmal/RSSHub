// internal/core/port/rss.go
package port

import (
	"context"
	"rsshub/internal/core/domain"
	"rsshub/internal/platform/utils"
	"time"
)

// FeedRepository defines storage operations
type FeedArticleRepository interface {
	CreateFeed(name, url string) (*domain.Feed, error)
	GetFeedByName(name string) (*domain.Feed, error)
	GetAllFeeds(limit int) ([]*domain.Feed, error)
	GetOldestFeeds(limit int) ([]*domain.Feed, error)
	UpdateFeedTimestamp(feedID utils.UUID) error
	DeleteFeed(name string) error
	CreateArticle(article *domain.Article) error
	GetArticlesByFeedName(feedName string, limit int) ([]*domain.Article, error)
	ArticleExists(link string) (bool, error)

	// Aggregator settings
	SetAggregatorSetting(key, value string) error
	GetAggregatorSetting(key string) (string, error)

	// Database locking
	TryLock(lockName string) (bool, error)
	ReleaseLock(lockName string) error
}

type Parser interface {
	FetchAndParse(url string) (*domain.ParsedRSSFeed, error)
	ValidateRSSURL(url string) error
}

type Aggregator interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
	SetInterval(newInterval time.Duration) error
	Resize(newWorkersCount int) error
	LoadSettingsFromDB() error
}
