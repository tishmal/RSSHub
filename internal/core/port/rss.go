package port

import (
	"context"
	"rsshub/internal/core/domain"
	"time"

	"github.com/google/uuid"
)

// FeedRepository defines storage operations
type FeedArticleRepository interface {
	CreateFeed(name, url string) (*domain.Feed, error)
	GetFeedByName(name string) (*domain.Feed, error)
	GetAllFeeds(limit int) ([]*domain.Feed, error)
	GetOldestFeeds(limit int) ([]*domain.Feed, error)
	UpdateFeedTimestamp(feedID uuid.UUID) error
	DeleteFeed(name string) error
	CreateArticle(article *domain.Article) error
	GetArticlesByFeedName(feedName string, limit int) ([]*domain.Article, error)
	ArticleExists(link string) (bool, error)
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
}
