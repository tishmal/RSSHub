package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"rsshub/internal/aggregator"
	"rsshub/internal/config"
	"rsshub/internal/database"
	"rsshub/internal/rss"
	"rsshub/pkg/logger"
)

// CLI представляет интерфейс командной строки
type CLI struct {
	db         *database.DB
	aggregator *aggregator.Aggregator
	config     *config.Config
}

// New создает новый CLI
func New(db *database.DB, cfg *config.Config) *CLI {
	// Создаем агрегатор с настройками по умолчанию
	agg := aggregator.New(db, cfg.Aggregator.DefaultInterval, cfg.Aggregator.DefaultWorkers)

	return &CLI{
		db:         db,
		aggregator: agg,
		config:     cfg,
	}
}

// Run запускает CLI и обрабатывает аргументы командной строки
func (c *CLI) Run(args []string) error {
	if len(args) < 2 {
		c.showHelp()
		return fmt.Errorf("no command provided")
	}

	command := args[1]

	switch command {
	case "fetch":
		return c.handleFetch()
	case "add":
		return c.handleAdd(args)
	case "set-interval":
		return c.handleSetInterval(args)
	case "set-workers":
		return c.handleSetWorkers(args)
	case "list":
		return c.handleList(args)
	case "delete":
		return c.handleDelete(args)
	case "articles":
		return c.handleArticles(args)
	case "--help", "-h", "help":
		c.showHelp()
		return nil
	default:
		c.showHelp()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleFetch запускает фоновый процесс получения RSS лент
func (c *CLI) handleFetch() error {
	// Проверяем, не запущен ли уже процесс
	if c.aggregator.IsRunning() {
		logger.Info("Background process is already running")
		return nil
	}

	// Запускаем агрегатор
	ctx := context.Background()
	if err := c.aggregator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start aggregator: %w", err)
	}

	// Ждем сигнала завершения (Ctrl+C)
	c.waitForShutdown()

	return nil
}

// handleAdd добавляет новую RSS ленту
func (c *CLI) handleAdd(args []string) error {
	var name, url string

	// Парсим аргументы
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return fmt.Errorf("--name requires a value")
			}
			name = args[i+1]
			i++
		case "--url":
			if i+1 >= len(args) {
				return fmt.Errorf("--url requires a value")
			}
			url = args[i+1]
			i++
		}
	}

	if name == "" || url == "" {
		return fmt.Errorf("both --name and --url are required")
	}

	// Валидируем RSS URL
	parser := rss.NewParser()
	if err := parser.ValidateRSSURL(url); err != nil {
		return fmt.Errorf("invalid RSS URL: %w", err)
	}

	// Создаем ленту в базе данных
	feed, err := c.db.CreateFeed(name, url)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return fmt.Errorf("feed with name '%s' already exists", name)
		}
		return fmt.Errorf("failed to create feed: %w", err)
	}

	logger.Success("Successfully added feed: %s (%s)", feed.Name, feed.URL)
	return nil
}

// handleSetInterval изменяет интервал получения лент
func (c *CLI) handleSetInterval(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("interval duration is required (e.g., '2m', '30s', '1h')")
	}

	durationStr := args[2]
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", durationStr)
	}

	if duration < time.Second {
		return fmt.Errorf("interval must be at least 1 second")
	}

	// Проверяем, запущен ли агрегатор
	if !c.aggregator.IsRunning() {
		return fmt.Errorf("background process is not running. Start it first with 'rsshub fetch'")
	}

	// Изменяем интервал
	return c.aggregator.SetInterval(duration)
}

// handleSetWorkers изменяет количество воркеров
func (c *CLI) handleSetWorkers(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("number of workers is required")
	}

	count, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("invalid workers count: %s", args[2])
	}

	if count <= 0 {
		return fmt.Errorf("workers count must be positive")
	}

	// Проверяем, запущен ли агрегатор
	if !c.aggregator.IsRunning() {
		return fmt.Errorf("background process is not running. Start it first with 'rsshub fetch'")
	}

	// Изменяем количество воркеров
	return c.aggregator.Resize(count)
}

// handleList показывает список RSS лент
func (c *CLI) handleList(args []string) error {
	var limit int

	// Парсим аргумент --num
	for i := 2; i < len(args); i++ {
		if args[i] == "--num" {
			if i+1 >= len(args) {
				return fmt.Errorf("--num requires a value")
			}
			var err error
			limit, err = strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid number: %s", args[i+1])
			}
			break
		}
	}

	// Получаем ленты из базы данных
	feeds, err := c.db.GetAllFeeds(limit)
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No RSS feeds found")
		return nil
	}

	fmt.Println("# Available RSS Feeds")
	fmt.Println()

	for i, feed := range feeds {
		fmt.Printf("%d. Name: %s\n", i+1, feed.Name)
		fmt.Printf("   URL: %s\n", feed.URL)
		fmt.Printf("   Added: %s\n", feed.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Println()
	}

	return nil
}

// handleDelete удаляет RSS ленту
func (c *CLI) handleDelete(args []string) error {
	var name string

	// Парсим аргумент --name
	for i := 2; i < len(args); i++ {
		if args[i] == "--name" {
			if i+1 >= len(args) {
				return fmt.Errorf("--name requires a value")
			}
			name = args[i+1]
			break
		}
	}

	if name == "" {
		return fmt.Errorf("--name is required")
	}

	// Удаляем ленту
	if err := c.db.DeleteFeed(name); err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	logger.Success("Successfully deleted feed: %s", name)
	return nil
}

// handleArticles показывает последние статьи из указанной ленты
func (c *CLI) handleArticles(args []string) error {
	var feedName string
	var limit int = 3 // По умолчанию

	// Парсим аргументы
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--feed-name":
			if i+1 >= len(args) {
				return fmt.Errorf("--feed-name requires a value")
			}
			feedName = args[i+1]
			i++
		case "--num":
			if i+1 >= len(args) {
				return fmt.Errorf("--num requires a value")
			}
			var err error
			limit, err = strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid number: %s", args[i+1])
			}
			i++
		}
	}

	if feedName == "" {
		return fmt.Errorf("--feed-name is required")
	}

	// Проверяем, существует ли лента
	_, err := c.db.GetFeedByName(feedName)
	if err != nil {
		return fmt.Errorf("feed not found: %s", feedName)
	}

	// Получаем статьи
	articles, err := c.db.GetArticlesByFeedName(feedName, limit)
	if err != nil {
		return fmt.Errorf("failed to get articles: %w", err)
	}

	if len(articles) == 0 {
		fmt.Printf("No articles found for feed: %s\n", feedName)
		return nil
	}

	fmt.Printf("Feed: %s\n\n", feedName)

	for i, article := range articles {
		date := article.PublishedAt.Format("2006-01-02")
		fmt.Printf("%d. [%s] %s\n", i+1, date, article.Title)
		fmt.Printf("   %s\n\n", article.Link)
	}

	return nil
}

// showHelp выводит справку по использованию CLI
func (c *CLI) showHelp() {
	fmt.Println(`Usage:
  rsshub COMMAND [OPTIONS]

Common Commands:
     add             add new RSS feed
     set-interval    set RSS fetch interval
     set-workers     set number of workers
     list            list available RSS feeds
     delete          delete RSS feed
     articles        show latest articles
     fetch           starts the background process that periodically fetches and processes RSS feeds using a worker pool

Examples:
     rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
     rsshub list --num 5
     rsshub delete --name "tech-crunch"
     rsshub articles --feed-name "tech-crunch" --num 5
     rsshub set-interval 2m
     rsshub set-workers 5
     rsshub fetch`)
}

// waitForShutdown ожидает сигнала завершения (Ctrl+C)
func (c *CLI) waitForShutdown() {
	// Создаем канал для получения сигналов ОС
	sigChan := make(chan os.Signal, 1)

	// Регистрируемся на получение SIGINT (Ctrl+C) и SIGTERM
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Press Ctrl+C to stop the aggregator...")

	// Ожидаем сигнал
	sig := <-sigChan
	logger.Info("Received signal: %v", sig)

	// Останавливаем агрегатор
	if err := c.aggregator.Stop(); err != nil {
		logger.Error("Error stopping aggregator: %v", err)
	}
}
