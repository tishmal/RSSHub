package main

import (
	"fmt"
	"os"
	"rsshub/internal/cli"

	"rsshub/internal/config"
	"rsshub/internal/storage"
)

func main() {
	cfg := config.Load()

	st := storage.NewMemory()

	if err := cli.Run(os.Args[1:], cfg, st); err != nil {
		fmt.Println(os.Stderr, err)
		os.Exit(1)
	}
}
