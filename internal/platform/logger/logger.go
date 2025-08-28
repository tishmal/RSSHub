package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger простой логгер для приложения
type Logger struct {
	*log.Logger
}

var defaultLogger *Logger

func init() {
	// Инициализируем логгер по умолчанию
	defaultLogger = &Logger{
		Logger: log.New(os.Stdout, "", 0), // Без стандартных флагов, добавим свои
	}
}

// Info выводит информационное сообщение
func Info(msg string, args ...interface{}) {
	defaultLogger.logWithLevel("INFO", msg, args...)
}

// Error выводит сообщение об ошибке
func Error(msg string, args ...interface{}) {
	defaultLogger.logWithLevel("ERROR", msg, args...)
}

// Debug выводит отладочное сообщение
func Debug(msg string, args ...interface{}) {
	defaultLogger.logWithLevel("DEBUG", msg, args...)
}

// Warn выводит предупреждение
func Warn(msg string, args ...interface{}) {
	defaultLogger.logWithLevel("WARN", msg, args...)
}

// logWithLevel форматирует и выводит сообщение с уровнем логирования
func (l *Logger) logWithLevel(level, msg string, args ...interface{}) {
	// Форматируем время в удобочитаемом виде
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Если есть аргументы, используем форматированный вывод
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// Выводим сообщение в формате: [ВРЕМЯ] УРОВЕНЬ: сообщение
	l.Logger.Printf("[%s] %s: %s", timestamp, level, msg)
}

// Fatal выводит критическую ошибку и завершает программу
func Fatal(msg string, args ...interface{}) {
	defaultLogger.logWithLevel("FATAL", msg, args...)
	os.Exit(1)
}

// Success выводит сообщение об успешном выполнении операции
func Success(msg string, args ...interface{}) {
	defaultLogger.logWithLevel("SUCCESS", msg, args...)
}
