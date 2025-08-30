// internal/core/service/aggregator.go
package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"rsshub/internal/core/domain"
	"rsshub/internal/core/port"
	"rsshub/internal/platform/logger"

	"rsshub/internal/platform/utils"
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

	// Менеджер настроек
	manager *AggregatorManager
}

// New создает новый агрегатор
func New(db port.FeedArticleRepository, parser port.Parser, defaultInterval time.Duration, defaultWorkers int) *Aggregator {
	return &Aggregator{
		db:           db,
		parser:       parser,
		interval:     defaultInterval,
		workersCount: defaultWorkers,
		isRunning:    false,
		manager:      NewAggregatorManager(db),
	}
}

// LoadSettingsFromDB загружает настройки агрегатора из базы данных
func (a *Aggregator) LoadSettingsFromDB() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Загружаем интервал
	if intervalStr, err := a.db.GetAggregatorSetting("interval"); err == nil {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			a.interval = interval
			logger.Info("Loaded interval from database: %v", interval)
		}
	}

	// Загружаем количество воркеров
	if workersStr, err := a.db.GetAggregatorSetting("workers"); err == nil {
		if workers, err := strconv.Atoi(workersStr); err == nil && workers > 0 {
			a.workersCount = workers
			logger.Info("Loaded workers count from database: %d", workers)
		}
	}

	return nil
}

// Start запускает фоновый процесс агрегации RSS лент
func (a *Aggregator) Start(ctx context.Context) error {
	a.runningMu.Lock()
	defer a.runningMu.Unlock()

	// Проверяем, не запущен ли уже процесс
	if a.isRunning {
		return fmt.Errorf("background process is already running")
	}

	// Загружаем настройки из базы данных
	if err := a.LoadSettingsFromDB(); err != nil {
		logger.Warn("Failed to load settings from database: %v", err)
	}

	// Создаем контекст для управления жизненным циклом
	a.ctx, a.cancel = context.WithCancel(ctx)

	// Инициализируем каналы и воркеры
	a.mu.RLock()
	workersCount := a.workersCount
	a.mu.RUnlock()

	a.jobs = make(chan *domain.Feed, workersCount*2) // Буферизированный канал

	// Запускаем воркеров
	for i := 0; i < workersCount; i++ {
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
		interval, workersCount)

	// Запускаем основной цикл агрегации
	go a.aggregationLoop()

	// Запускаем мониторинг изменений настроек
	go a.manager.StartMonitoring(a.ctx, a)

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

// SetInterval динамически изменяет интервал получения лент (только для запущенного агрегатора)
func (a *Aggregator) SetInterval(newInterval time.Duration) error {
	a.runningMu.RLock()
	isRunning := a.isRunning
	a.runningMu.RUnlock()

	if !isRunning {
		return fmt.Errorf("aggregator is not running")
	}

	a.mu.Lock()
	oldInterval := a.interval

	// Проверяем, действительно ли интервал изменился
	if oldInterval == newInterval {
		a.mu.Unlock()
		return nil // Нет изменений
	}

	a.interval = newInterval
	a.mu.Unlock()

	// Перезапускаем тикер с новым интервалом
	if a.ticker != nil {
		a.ticker.Stop()
		a.ticker = time.NewTicker(newInterval)
	}

	logger.Success("Interval of fetching feeds changed from %v to %v (applied dynamically)", oldInterval, newInterval)
	return nil
}

// Resize динамически изменяет количество воркеров (только для запущенного агрегатора)
func (a *Aggregator) Resize(newWorkersCount int) error {
	if newWorkersCount <= 0 {
		return fmt.Errorf("workers count must be positive")
	}

	a.runningMu.RLock()
	isRunning := a.isRunning
	a.runningMu.RUnlock()

	a.mu.Lock()
	oldCount := a.workersCount

	// Проверяем, действительно ли количество воркеров изменилось
	if oldCount == newWorkersCount {
		a.mu.Unlock()
		return nil // Нет изменений
	}

	if !isRunning {
		// Если агрегатор не запущен, просто обновляем значение
		a.workersCount = newWorkersCount
		a.mu.Unlock()
		logger.Success("Number of workers changed from %d to %d", oldCount, newWorkersCount)
		return nil
	}

	// Если агрегатор запущен, нужно управлять воркерами
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
	a.mu.Unlock()

	logger.Success("Number of workers changed from %d to %d (applied dynamically)", oldCount, newWorkersCount)

	return nil
}

// aggregationLoop запускает основной цикл агрегации
func (a *Aggregator) aggregationLoop() {
	settingsTicker := time.NewTicker(10 * time.Second)
	defer settingsTicker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return

		case <-a.ticker.C:
			go a.fetchFeeds()

		case <-settingsTicker.C:
			logger.Info("Checking DB for settings changes...")
			if err := a.manager.CheckAndApplyChanges(a); err != nil {
				logger.Error("Failed to apply settings changes: %v", err)
			}
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
		uuid, err := utils.NewUUID()
		if err != nil {
			logger.Error("UUID error: %v", err)
			continue
		}
		// Создаем новую статью
		article := &domain.Article{
			ID:          uuid,
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
