package storage

import (
	"database/sql"
	"fmt"
	"time"

	"rsshub/internal/core/domain"
	"rsshub/internal/platform/logger"
	"rsshub/internal/platform/utils"

	_ "github.com/lib/pq" // PostgreSQL драйвер
)

// DB оборачивает sql.DB и предоставляет методы для работы с нашими моделями
type DB struct {
	*sql.DB
}

// New создает новое подключение к базе данных
func New(dsn string) (*DB, error) {
	// Открываем соединение с PostgreSQL
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(25)                 // Максимум открытых соединений
	db.SetMaxIdleConns(25)                 // Максимум неактивных соединений
	db.SetConnMaxLifetime(5 * time.Minute) // Время жизни соединения

	logger.Info("Successfully connected to PostgreSQL database")

	return &DB{DB: db}, nil
}

// CreateFeed создает новую RSS ленту в базе данных
func (db *DB) CreateFeed(name, url string) (*domain.Feed, error) {
	uuid, _err := utils.NewUUID()
	if _err != nil {
		return nil, _err
	}

	feed := &domain.Feed{
		ID:        uuid,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		URL:       url,
	}

	// SQL запрос для вставки новой ленты
	query := `
		INSERT INTO feeds (id, created_at, updated_at, name, url)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, feed.ID.String(), feed.CreatedAt, feed.UpdatedAt, feed.Name, feed.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	logger.Info("Created new feed: %s (%s)", name, url)
	return feed, nil
}

// GetFeedByName получает ленту по имени
func (db *DB) GetFeedByName(name string) (*domain.Feed, error) {
	feed := &domain.Feed{}

	query := `
		SELECT id, created_at, updated_at, name, url 
		FROM feeds 
		WHERE name = $1`
	var idFeed string
	err := db.QueryRow(query, name).
		Scan(&idFeed, &feed.CreatedAt, &feed.UpdatedAt, &feed.Name, &feed.URL)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	feed.ID, err = utils.ParseUUID(idFeed)
	if err != nil {
		return nil, fmt.Errorf("UUID error: %v", err)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	return feed, nil
}

// GetAllFeeds получает все ленты, опционально ограничивая количество
func (db *DB) GetAllFeeds(limit int) ([]*domain.Feed, error) {
	var query string
	var args []interface{}

	if limit > 0 {
		// С ограничением количества, сортируем по дате создания (новые сначала)
		query = `
			SELECT id, created_at, updated_at, name, url 
			FROM feeds 
			ORDER BY created_at DESC 
			LIMIT $1`
		args = append(args, limit)
	} else {
		// Без ограничений
		query = `
			SELECT id, created_at, updated_at, name, url 
			FROM feeds 
			ORDER BY created_at DESC`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*domain.Feed
	var idFeed string
	for rows.Next() {
		feed := &domain.Feed{}
		err := rows.Scan(&idFeed, &feed.CreatedAt, &feed.UpdatedAt, &feed.Name, &feed.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}
		feed.ID, err = utils.ParseUUID(idFeed)
		if err != nil {
			return nil, fmt.Errorf("failed to UIID feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// GetOldestFeeds получает N самых устаревших лент для обновления
func (db *DB) GetOldestFeeds(limit int) ([]*domain.Feed, error) {
	query := `
		SELECT id, created_at, updated_at, name, url 
		FROM feeds 
		ORDER BY updated_at ASC 
		LIMIT $1`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get oldest feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*domain.Feed
	var idFeed string
	for rows.Next() {
		feed := &domain.Feed{}
		err := rows.Scan(&idFeed, &feed.CreatedAt, &feed.UpdatedAt, &feed.Name, &feed.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}

		feed.ID, _ = utils.ParseUUID(idFeed)

		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// UpdateFeedTimestamp обновляет время последнего обновления ленты
func (db *DB) UpdateFeedTimestamp(feedID utils.UUID) error {
	query := `UPDATE feeds SET updated_at = $1 WHERE id = $2`

	_, err := db.Exec(query, time.Now(), feedID.String())
	if err != nil {
		return fmt.Errorf("failed to update feed timestamp: %w", err)
	}

	return nil
}

// DeleteFeed удаляет ленту по имени
func (db *DB) DeleteFeed(name string) error {
	// Сначала проверяем, существует ли лента
	_, err := db.GetFeedByName(name)
	if err != nil {
		return err // Лента не найдена или другая ошибка
	}

	query := `DELETE FROM feeds WHERE name = $1`

	result, err := db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	// Проверяем, что что-то было удалено
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no feed was deleted")
	}

	logger.Info("Deleted feed: %s", name)
	return nil
}

// CreateArticle создает новую статью в базе данных
func (db *DB) CreateArticle(article *domain.Article) error {
	// Генерируем ID если его нет
	if article.ID.String() == "" {
		if uuid, err := utils.NewUUID(); err != nil {
			return fmt.Errorf("UUID error")
		} else {
			article.ID = uuid
		}
	}

	// Устанавливаем текущее время если не установлено
	if article.CreatedAt.IsZero() {
		article.CreatedAt = time.Now()
	}
	if article.UpdatedAt.IsZero() {
		article.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO articles (id, created_at, updated_at, title, link, published_at, description, feed_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (link) DO NOTHING` // Игнорируем дубликаты по URL

	_, err := db.Exec(query,
		article.ID.String(), article.CreatedAt, article.UpdatedAt,
		article.Title, article.Link, article.PublishedAt,
		article.Description, article.FeedID.String())

	if err != nil {
		return fmt.Errorf("failed to create article: %w", err)
	}

	return nil
}

// GetArticlesByFeedName получает статьи для конкретной ленты по имени
func (db *DB) GetArticlesByFeedName(feedName string, limit int) ([]*domain.Article, error) {
	if limit <= 0 {
		limit = 3 // Значение по умолчанию
	}

	query := `
		SELECT a.id, a.created_at, a.updated_at, a.title, a.link, a.published_at, a.description, a.feed_id
		FROM articles a
		JOIN feeds f ON a.feed_id = f.id
		WHERE f.name = $1
		ORDER BY a.published_at DESC
		LIMIT $2`

	rows, err := db.Query(query, feedName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}
	defer rows.Close()

	var articles []*domain.Article
	var articleID string
	var feedID string

	for rows.Next() {
		article := &domain.Article{}
		err := rows.Scan(
			&articleID, &article.CreatedAt, &article.UpdatedAt,
			&article.Title, &article.Link, &article.PublishedAt,
			&article.Description, &feedID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		article.ID, err = utils.ParseUUID(articleID)
		if err != nil {
			return nil, fmt.Errorf("failed parsing article ID: %w", err)
		}

		article.FeedID, err = utils.ParseUUID(feedID)
		if err != nil {
			return nil, fmt.Errorf("failed parsing feed ID: %w", err)
		}

		articles = append(articles, article)
	}

	return articles, nil
}

// ArticleExists проверяет, существует ли статья с данным URL
func (db *DB) ArticleExists(link string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE link = $1)`

	err := db.QueryRow(query, link).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check article existence: %w", err)
	}

	return exists, nil
}
