package main

import (
	"os"

	"rsshub/internal/cli"
	"rsshub/internal/config"
	"rsshub/internal/database"
	"rsshub/pkg/logger"
)

func main() {
	// Загружаем конфигурацию из переменных окружения
	cfg := config.Load()

	// Подключаемся к базе данных
	db, err := database.New(cfg.Database.GetDSN())
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Запускаем миграции
	if err := db.RunMigrations(); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}

	// Создаем CLI и запускаем команду
	cliApp := cli.New(db, cfg)

	if err := cliApp.Run(os.Args); err != nil {
		logger.Error("Command failed: %v", err)
		os.Exit(1)
	}
}
