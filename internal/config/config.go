package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DefaultInterval time.Duration
	DefaultWorkers  int
	ControlAddr     string
}

func Load() Config {
	interval := 3 * time.Minute
	if v := os.Getenv("CLI_APP_TIMER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	workers := 3
	if v := os.Getenv("CLI_APP_WORKERS_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workers = n
		}
	}

	addr := os.Getenv("CLI_APP_CONTROL_ADDR")
	if addr == "" {
		addr = "127.0.0.1:7070"
	}

	return Config{
		DefaultInterval: interval,
		DefaultWorkers:  workers,
		ControlAddr:     addr,
	}
}
