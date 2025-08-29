// internal/core/service/manager.go
package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"rsshub/internal/core/port"
	"rsshub/internal/platform/logger"
)

// AggregatorManager управляет общими настройками агрегатора через базу данных
type AggregatorManager struct {
	db port.FeedArticleRepository
}

// NewAggregatorManager создает новый менеджер агрегатора
func NewAggregatorManager(db port.FeedArticleRepository) *AggregatorManager {
	return &AggregatorManager{
		db: db,
	}
}

// SetInterval устанавливает интервал и уведомляет запущенный агрегатор
func (m *AggregatorManager) SetInterval(duration time.Duration) error {
	// Сохраняем в базу данных
	if err := m.db.SetAggregatorSetting("interval", duration.String()); err != nil {
		return fmt.Errorf("failed to save interval to database: %w", err)
	}

	// Устанавливаем флаг для уведомления агрегатора об изменениях
	if err := m.db.SetAggregatorSetting("settings_changed", "true"); err != nil {
		logger.Warn("Failed to set settings change flag: %v", err)
	}

	logger.Success("Interval set to %v (will be applied to running aggregator)", duration)
	return nil
}

// SetWorkers устанавливает количество воркеров и уведомляет запущенный агрегатор
func (m *AggregatorManager) SetWorkers(count int) error {
	// Сохраняем в базу данных
	if err := m.db.SetAggregatorSetting("workers", strconv.Itoa(count)); err != nil {
		return fmt.Errorf("failed to save workers count to database: %w", err)
	}

	// Устанавливаем флаг для уведомления агрегатора об изменениях
	if err := m.db.SetAggregatorSetting("settings_changed", "true"); err != nil {
		logger.Warn("Failed to set settings change flag: %v", err)
	}

	logger.Success("Workers count set to %d (will be applied to running aggregator)", count)
	return nil
}

// CheckAndApplyChanges проверяет изменения настроек и применяет их к агрегатору
func (m *AggregatorManager) CheckAndApplyChanges(aggregator port.Aggregator) error {
	// Проверяем, есть ли изменения настроек
	changed, err := m.db.GetAggregatorSetting("settings_changed")
	if err != nil || changed != "true" {
		return nil // Нет изменений
	}

	logger.Info("Detected settings changes, applying...")

	// Сбрасываем флаг изменений СНАЧАЛА, чтобы избежать повторного применения
	if err := m.db.SetAggregatorSetting("settings_changed", "false"); err != nil {
		logger.Warn("Failed to reset settings change flag: %v", err)
	}

	// Загружаем новые настройки из базы данных
	var newInterval time.Duration
	var newWorkers int

	if intervalStr, err := m.db.GetAggregatorSetting("interval"); err == nil {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			newInterval = interval
		}
	}

	if workersStr, err := m.db.GetAggregatorSetting("workers"); err == nil {
		if workers, err := strconv.Atoi(workersStr); err == nil && workers > 0 {
			newWorkers = workers
		}
	}

	// Применяем изменения к агрегатору
	if newInterval > 0 {
		if err := aggregator.SetInterval(newInterval); err != nil {
			logger.Error("Failed to apply interval change: %v", err)
		}
	}

	if newWorkers > 0 {
		if err := aggregator.Resize(newWorkers); err != nil {
			logger.Error("Failed to apply workers change: %v", err)
		}
	}

	return nil
}

// StartMonitoring запускает мониторинг изменений настроек
func (m *AggregatorManager) StartMonitoring(ctx context.Context, aggregator port.Aggregator) {
	ticker := time.NewTicker(10 * time.Second) // Увеличиваем интервал до 10 секунд
	defer ticker.Stop()

	logger.Debug("Settings monitoring started (checking every 10 seconds)")

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Settings monitoring stopped")
			return
		case <-ticker.C:
			if err := m.CheckAndApplyChanges(aggregator); err != nil {
				logger.Error("Error checking settings changes: %v", err)
			}
		}
	}
}
