package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/signal"
	"rsshub/internal/aggregator"
	"rsshub/internal/config"
	"rsshub/internal/storage"
	"syscall"
)

func Run(args []string, cfg config.Config, st storage.Storage) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "fetch":
		return cmdFetch(cfg, st)
	}

}

func printHelp() {
	fmt.Println(`Usage:
rsshub COMMAND [OPTIONS]


Commands:
add add new RSS feed
set-interval set RSS fetch interval (e.g. rsshub set-interval 2m)
set-workers set number of workers (e.g. rsshub set-workers 5)
list list available RSS feeds
delete delete RSS feed by name
articles show latest articles by feed name
fetch start background process (ticker + workers)
`)
}

func cmdFetch(cfg config.Config, st storage.Storage) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	aggr := aggregator.New(st, cfg.DefaultInterval, cfg.DefaultWorkers)
	if err := aggr.Start(ctx); err != nil {
		return err
	}

	// start control server
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
		fmt.Printf("\n%d. Name: %s\n URL: %s\n Added: %s\n", i+1, f.Name, f.URL, f.CreatedAt.Format("2006-01-02 15:04"))
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
		fmt.Printf("%d. [%s] %s\n %s\n\n", i+1, a.PublishedAt.Format("2006-01-02"), a.Title, a.Link)
	}
	return nil
}
