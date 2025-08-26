package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rsshub/internal/aggregator"
	"rsshub/internal/config"
	"rsshub/internal/storage"
)

func Run(args []string, cfg config.Config, st storage.Storage) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "fetch":
		return cmdFetch(cfg, st)
	case "set-interval":
		return cmdSetInterval(cfg, args[1:])
	case "set-workers":
		return cmdSetWorkers(cfg, args[1:])
	case "add":
		return cmdAdd(st, args[1:])
	case "list":
		return cmdList(st, args[1:])
	case "delete":
		return cmdDelete(st, args[1:])
	case "articles":
		return cmdArticles(st, args[1:])
	case "--help", "-h", "help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printHelp() {
	fmt.Println(`Usage:
  rsshub COMMAND [OPTIONS]

Commands:
  add             add new RSS feed
  set-interval    set RSS fetch interval (e.g. rsshub set-interval 2m)
  set-workers     set number of workers   (e.g. rsshub set-workers 5)
  list            list available RSS feeds
  delete          delete RSS feed by name
  articles        show latest articles by feed name
  fetch           start background process (ticker + workers)
`)
}

func cmdFetch(cfg config.Config, st storage.Storage) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	aggr := aggregator.New(st, cfg.DefaultInterval, cfg.DefaultWorkers)
	if err := aggr.Start(ctx); err != nil {
		return err
	}

	// Стартуем control-сервер
	ctrl := aggregator.NewControlServer(cfg.ControlAddr, aggr)
	if err := ctrl.Start(); err != nil {
		return err
	}
	defer ctrl.Stop()

	fmt.Printf("The background process for fetching feeds has started (interval = %s, workers = %d)\n", cfg.DefaultInterval, cfg.DefaultWorkers)
	<-ctx.Done()
	_ = aggr.Stop()
	fmt.Println("Graceful shutdown: aggregator stopped")
	return nil
}

func cmdSetInterval(cfg config.Config, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: rsshub set-interval <duration>")
	}
	d, err := time.ParseDuration(args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	conn, err := net.Dial("tcp", cfg.ControlAddr)
	if err != nil {
		return fmt.Errorf("background process is not running or control address unavailable (%s)", cfg.ControlAddr)
	}
	defer conn.Close()

	if err := aggregator.SendControlSetInterval(conn, d); err != nil {
		return err
	}
	return nil
}

func cmdSetWorkers(cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("set-workers", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	var count int
	if len(args) == 0 {
		return errors.New("usage: rsshub set-workers <count>")
	}
	if n, err := fmt.Sscanf(args[0], "%d", &count); n != 1 || err != nil || count <= 0 {
		return errors.New("count must be positive integer")
	}

	conn, err := net.Dial("tcp", cfg.ControlAddr)
	if err != nil {
		return fmt.Errorf("background process is not running or control address unavailable (%s)", cfg.ControlAddr)
	}
	defer conn.Close()

	if err := aggregator.SendControlSetWorkers(conn, count); err != nil {
		return err
	}
	return nil
}

func cmdAdd(st storage.Storage, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	var name, url string
	fs.StringVar(&name, "name", "", "feed name")
	fs.StringVar(&url, "url", "", "feed url")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" || url == "" {
		return errors.New("usage: rsshub add --name <name> --url <url>")
	}
	if err := st.AddFeed(name, url); err != nil {
		return err
	}
	fmt.Println("Feed added:", name)
	return nil
}

func cmdList(st storage.Storage, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	var num int
	fs.IntVar(&num, "num", 0, "limit number of feeds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	feeds, err := st.ListFeeds(num)
	if err != nil {
		return err
	}
	fmt.Println("# Available RSS Feeds")
	for i, f := range feeds {
		fmt.Printf("\n%d. Name: %s\n   URL: %s\n   Added: %s\n", i+1, f.Name, f.URL, f.CreatedAt.Format("2006-01-02 15:04"))
	}
	return nil
}

func cmdDelete(st storage.Storage, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	var name string
	fs.StringVar(&name, "name", "", "feed name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" {
		return errors.New("usage: rsshub delete --name <name>")
	}
	if err := st.DeleteFeed(name); err != nil {
		return err
	}
	fmt.Println("Feed deleted:", name)
	return nil
}

func cmdArticles(st storage.Storage, args []string) error {
	fs := flag.NewFlagSet("articles", flag.ContinueOnError)
	var name string
	var num int
	fs.StringVar(&name, "feed-name", "", "feed name")
	fs.IntVar(&num, "num", 3, "number of articles")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" {
		return errors.New("usage: rsshub articles --feed-name <name> [--num N]")
	}
	arts, err := st.RecentArticlesByFeed(name, num)
	if err != nil {
		return err
	}
	fmt.Printf("Feed: %s\n\n", name)
	for i, a := range arts {
		fmt.Printf("%d. [%s] %s\n   %s\n\n", i+1, a.PublishedAt.Format("2006-01-02"), a.Title, a.Link)
	}
	return nil
}
