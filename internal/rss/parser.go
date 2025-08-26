package rss

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"time"
)

// Структуры под парсинг XML

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func FetchAndParse(ctx context.Context, client *http.Client, url string) (RSSFeed, error) {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return RSSFeed{}, err
	}
	req.Header.Set("User-Agent", "rsshub/1.0 (+https://example.local)")
	resp, err := client.Do(req)
	if err != nil {
		return RSSFeed{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return RSSFeed{}, io.ErrUnexpectedEOF
	}
	var feed RSSFeed
	dec := xml.NewDecoder(resp.Body)
	if err := dec.Decode(&feed); err != nil {
		return RSSFeed{}, err
	}
	return feed, nil
}

// ToArticles — конвертация RSSItem в доменную модель storage.ArticleInput
func ToArticles(feed RSSFeed) []ArticleInput {
	res := make([]ArticleInput, 0, len(feed.Channel.Item))
	for _, it := range feed.Channel.Item {
		// Парсим pubDate в time.Time, если не получилось — используем zero time
		var t time.Time
		if it.PubDate != "" {
			if tt, err := time.Parse(time.RFC1123Z, it.PubDate); err == nil {
				t = tt
			} else if tt, err := time.Parse(time.RFC1123, it.PubDate); err == nil {
				t = tt
			}
		}
		res = append(res, ArticleInput{
			Title:       it.Title,
			Link:        it.Link,
			Description: it.Description,
			PublishedAt: t,
		})
	}
	return res
}

// ArticleInput — лёгкая прослойка, чтобы не тянуть storage внутрь пакета rss.
// На реальной реализации мы будем использовать storage.ArticleInput.
type ArticleInput struct {
	Title       string
	Link        string
	Description string
	PublishedAt time.Time
}
