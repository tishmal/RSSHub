package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"rsshub/internal/core/domain"
	"rsshub/internal/core/port"
	"rsshub/internal/platform/logger"

	"github.com/google/uuid"
)

// Aggregator управляет фоновым процессом получения RSS лент
type Aggregator struct {
	db     port.FeedArticleRepository // База данных
	parser port.Parser                // RSS парсер

	// Настройки воркеров и интервала
	mu           sync.RWMutex  // Мьютекс для безопасного доступа к настройкам
	interval     time.Duration // Интервал между запусками
	workersCount int           // Количество воркеров

	// Управление жизненным циклом
	ctx    context.Context    // Контекст для graceful shutdown
	cancel context.CancelFunc // Функция отмены контекста
	ticker *time.Ticker       // Таймер для периодических запусков

	// Каналы для координации воркеров
	jobs     chan *domain.Feed // Канал заданий для воркеров
	workerWg sync.WaitGroup    // WaitGroup для ожидания завершения воркеров

	// Состояние
	isRunning bool         // Флаг запущенного состояния
	runningMu sync.RWMutex // Мьютекс для проверки состояния
}

// New создает новый агрегатор
func New(db port.FeedArticleRepository, parser port.Parser, defaultInterval time.Duration, defaultWorkers int) *Aggregator {
	return &Aggregator{
		db:           db,
		parser:       parser,
		interval:     defaultInterval,
		workersCount: defaultWorkers,
		isRunning:    false,
	}
}

// Start запускает фоновый процесс агрегации RSS лент
func (a *Aggregator) Start(ctx context.Context) error {
	a.runningMu.Lock()
	defer a.runningMu.Unlock()

	// Проверяем, не запущен ли уже процесс
	if a.isRunning {
		return fmt.Errorf("background process is already running")
	}

	// Создаем контекст для управления жизненным циклом
	a.ctx, a.cancel = context.WithCancel(ctx)

	// Инициализируем каналы и воркеры
	a.jobs = make(chan *domain.Feed, a.workersCount*2) // Буферизированный канал

	// Запускаем воркеров
	for i := 0; i < a.workersCount; i++ {
		a.workerWg.Add(1)
		go a.worker(i + 1)
	}

	// Создаем и запускаем тикер
	a.mu.RLock()
	interval := a.interval
	a.mu.RUnlock()

	a.ticker = time.NewTicker(interval)
	a.isRunning = true

	logger.Success("The background process for fetching feeds has started (interval = %v, workers = %d)",
		interval, a.workersCount)

	// Запускаем основной цикл агрегации
	go a.aggregationLoop()

	// Делаем первый запуск сразу, не дожидаясь тикера
	go a.fetchFeeds()

	return nil
}

// Stop останавливает фоновый процесс gracefully
func (a *Aggregator) Stop() error {
	a.runningMu.Lock()
	defer a.runningMu.Unlock()

	if !a.isRunning {
		return fmt.Errorf("background process is not running")
	}

	logger.Info("Stopping background aggregation process...")

	// Останавливаем тикер
	if a.ticker != nil {
		a.ticker.Stop()
	}

	// Отменяем контекст
	if a.cancel != nil {
		a.cancel()
	}

	// Закрываем канал заданий
	if a.jobs != nil {
		close(a.jobs)
	}

	// Ждем завершения всех воркеров
	a.workerWg.Wait()

	a.isRunning = false
	logger.Success("Graceful shutdown: aggregator stopped")

	return nil
}

// IsRunning проверяет, запущен ли агрегатор
func (a *Aggregator) IsRunning() bool {
	a.runningMu.RLock()
	defer a.runningMu.RUnlock()
	return a.isRunning
}

// SetInterval динамически изменяет интервал получения лент
func (a *Aggregator) SetInterval(newInterval time.Duration) error {
	a.mu.Lock()
	oldInterval := a.interval
	a.interval = newInterval

	fmt.Println(a.interval) // check

	a.mu.Unlock()

	a.runningMu.RLock()
	isRunning := a.isRunning
	a.runningMu.RUnlock()

	// Если агрегатор запущен, нужно перезапустить тикер
	if isRunning {
		if a.ticker != nil {
			a.ticker.Stop()
			a.ticker = time.NewTicker(newInterval)
		}
		logger.Success("Interval of fetching feeds changed from %v to %v", oldInterval, newInterval)
	}

	return nil
}

// Resize динамически изменяет количество воркеров
func (a *Aggregator) Resize(newWorkersCount int) error {
	if newWorkersCount <= 0 {
		return fmt.Errorf("workers count must be positive")
	}

	a.mu.Lock()
	oldCount := a.workersCount
	a.mu.Unlock()

	a.runningMu.RLock()
	isRunning := a.isRunning
	a.runningMu.RUnlock()

	if !isRunning {
		// Если агрегатор не запущен, просто обновляем значение
		a.mu.Lock()
		a.workersCount = newWorkersCount
		a.mu.Unlock()
		return nil
	}

	// Если агрегатор запущен, нужно управлять воркерами
	a.mu.Lock()
	defer a.mu.Unlock()

	if newWorkersCount > a.workersCount {
		// Добавляем новых воркеров
		for i := a.workersCount; i < newWorkersCount; i++ {
			a.workerWg.Add(1)
			go a.worker(i + 1)
		}
	}
	// Если нужно уменьшить количество воркеров, они завершатся естественным образом
	// когда закончатся задания или при следующем shutdown

	a.workersCount = newWorkersCount
	logger.Success("Number of workers changed from %d to %d", oldCount, newWorkersCount)

	return nil
}

// aggregationLoop основной цикл агрегации
func (a *Aggregator) aggregationLoop() {
	for {
		select {
		case <-a.ctx.Done():
			// Контекст отменен, завершаем цикл
			return
		case <-a.ticker.C:
			// Время для очередного получения лент
			go a.fetchFeeds()
		}
	}
}

// fetchFeeds получает устаревшие ленты и распределяет их между воркерами
func (a *Aggregator) fetchFeeds() {
	logger.Info("-----------------------------")
	logger.Info("Starting feeds fetch cycle...")
	logger.Info("-----------------------------")
	// Получаем самые устаревшие ленты
	a.mu.RLock()
	workersCount := a.workersCount
	a.mu.RUnlock()

	feeds, err := a.db.GetOldestFeeds(workersCount)
	if err != nil {
		logger.Error("Failed to get feeds: %v", err)
		return
	}

	if len(feeds) == 0 {
		logger.Info("No feeds to process")
		return
	}

	logger.Info("Found %d feeds to process", len(feeds))

	// Отправляем ленты воркерам
	for _, feed := range feeds {
		select {
		case a.jobs <- feed:
			// Задание отправлено
		case <-a.ctx.Done():
			// Контекст отменен
			return
		default:
			// Канал заполнен, пропускаем эту ленту
			logger.Warn("Workers are busy, skipping feed: %s", feed.Name)
		}
	}
}

// worker обрабатывает ленты из канала заданий
func (a *Aggregator) worker(id int) {
	defer a.workerWg.Done()

	logger.Debug("Worker %d started", id)

	for {
		select {
		case feed, ok := <-a.jobs:
			if !ok {
				// Канал закрыт, завершаем воркер
				logger.Debug("Worker %d stopped (channel closed)", id)
				return
			}

			// Обрабатываем ленту
			a.processFeed(id, feed)

		case <-a.ctx.Done():
			// Контекст отменен, завершаем воркер
			logger.Debug("Worker %d stopped (context cancelled)", id)
			return
		}
	}
}

// processFeed обрабатывает одну RSS ленту
func (a *Aggregator) processFeed(workerID int, feed *domain.Feed) {
	logger.Info("Worker %d processing feed: %s (%s)", workerID, feed.Name, feed.URL)

	// Получаем и парсим RSS ленту
	parsedFeed, err := a.parser.FetchAndParse(feed.URL)
	if err != nil {
		logger.Error("Worker %d failed to fetch feed %s: %v", workerID, feed.Name, err)
		return
	}

	// Сохраняем новые статьи
	newArticles := 0
	for _, item := range parsedFeed.Items {
		// Проверяем, существует ли уже эта статья
		exists, err := a.db.ArticleExists(item.Link)
		if err != nil {
			logger.Error("Worker %d failed to check article existence: %v", workerID, err)
			continue
		}

		if exists {
			// Статья уже существует, пропускаем
			continue
		}

		// Создаем новую статью
		article := &domain.Article{
			ID:          uuid.New(),
			Title:       item.Title,
			Link:        item.Link,
			PublishedAt: item.PublishedAt,
			Description: item.Description,
			FeedID:      feed.ID,
		}

		if err := a.db.CreateArticle(article); err != nil {
			logger.Error("Worker %d failed to save article '%s': %v", workerID, item.Title, err)
			continue
		}

		newArticles++
	}

	// Обновляем timestamp ленты
	if err := a.db.UpdateFeedTimestamp(feed.ID); err != nil {
		logger.Error("Worker %d failed to update feed timestamp: %v", workerID, err)
	}

	logger.Success("Worker %d completed feed %s: %d new articles", workerID, feed.Name, newArticles)
}
