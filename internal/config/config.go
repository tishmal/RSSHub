package config

import (
	"os"
	"strconv"
	"time"
)

// Config хранит конфигурацию приложения
type Config struct {
	// Настройки базы данных PostgreSQL
	Database DatabaseConfig
	// Настройки агрегатора RSS
	Aggregator AggregatorConfig
}

// DatabaseConfig содержит параметры подключения к БД
type DatabaseConfig struct {
	Host     string // Хост PostgreSQL
	Port     int    // Порт PostgreSQL
	User     string // Имя пользователя
	Password string // Пароль
	DBName   string // Имя базы данных
}

// AggregatorConfig содержит настройки для фонового агрегатора
type AggregatorConfig struct {
	DefaultInterval time.Duration // Интервал по умолчанию для получения лент
	DefaultWorkers  int           // Количество воркеров по умолчанию
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "changeme"),
			DBName:   getEnv("POSTGRES_DBNAME", "rsshub"),
		},
		Aggregator: AggregatorConfig{
			DefaultInterval: getEnvDuration("CLI_APP_TIMER_INTERVAL", 3*time.Minute),
			DefaultWorkers:  getEnvInt("CLI_APP_WORKERS_COUNT", 3),
		},
	}
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt получает целочисленное значение переменной окружения
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvDuration получает значение времени из переменной окружения
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetDSN возвращает строку подключения к PostgreSQL
func (d *DatabaseConfig) GetDSN() string {
	return "host=" + d.Host +
		" port=" + strconv.Itoa(d.Port) +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.DBName +
		" sslmode=disable"
}
