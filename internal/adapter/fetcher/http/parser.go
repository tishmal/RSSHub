package httpfetcher

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rsshub/internal/core/domain"
	"rsshub/internal/core/port"
	"rsshub/internal/platform/logger"
)

// Parser отвечает за получение и парсинг RSS лент
type Parser struct {
	client *http.Client
}

// NewParser создает новый RSS парсер
func NewParser() port.Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 30 * time.Second, // Таймаут для HTTP запросов
		},
	}
}

// FetchAndParse получает RSS ленту по URL и парсит её
func (p *Parser) FetchAndParse(url string) (*domain.ParsedRSSFeed, error) {
	logger.Info("Fetching RSS feed: %s", url)

	// Делаем HTTP запрос к RSS ленте
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Проверяем статус код ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RSS feed returned status %d: %s", resp.StatusCode, url)
	}

	// Парсим XML в структуру RSS
	var rssFeed domain.RSSFeed
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&rssFeed); err != nil {
		return nil, fmt.Errorf("failed to parse RSS XML from %s: %w", url, err)
	}

	// Конвертируем сырую RSS структуру в нашу обработанную версию
	parsed, err := p.convertToParsedFeed(&rssFeed)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RSS feed %s: %w", url, err)
	}

	logger.Info("Successfully parsed RSS feed: %s (%d items)", url, len(parsed.Items))
	return parsed, nil
}

// convertToParsedFeed конвертирует сырую RSS структуру в обработанную
func (p *Parser) convertToParsedFeed(rssFeed *domain.RSSFeed) (*domain.ParsedRSSFeed, error) {
	parsed := &domain.ParsedRSSFeed{
		Title:       rssFeed.Channel.Title,
		Link:        rssFeed.Channel.Link,
		Description: rssFeed.Channel.Description,
		Items:       make([]domain.ParsedRSSItem, 0, len(rssFeed.Channel.Items)),
	}

	// Обрабатываем каждый элемент RSS ленты
	for _, item := range rssFeed.Channel.Items {
		parsedItem, err := p.convertRSSItem(&item)
		if err != nil {
			// Логируем ошибку, но продолжаем обработку остальных элементов
			logger.Warn("Failed to parse RSS item '%s': %v", item.Title, err)
			continue
		}
		parsed.Items = append(parsed.Items, *parsedItem)
	}

	return parsed, nil
}

// convertRSSItem конвертирует отдельный элемент RSS в нашу структуру
func (p *Parser) convertRSSItem(item *domain.RSSItem) (*domain.ParsedRSSItem, error) {
	parsed := &domain.ParsedRSSItem{
		Title:       strings.TrimSpace(item.Title),
		Link:        strings.TrimSpace(item.Link),
		Description: strings.TrimSpace(item.Description),
	}

	// Парсим дату публикации
	if item.PubDate != "" {
		publishedAt, err := p.parseRSSDate(item.PubDate)
		if err != nil {
			logger.Warn("Failed to parse date '%s' for item '%s': %v", item.PubDate, item.Title, err)
			// Используем текущее время как fallback
			parsed.PublishedAt = time.Now()
		} else {
			parsed.PublishedAt = publishedAt
		}
	} else {
		// Если дата не указана, используем текущее время
		parsed.PublishedAt = time.Now()
	}

	// Валидируем обязательные поля
	if parsed.Title == "" {
		return nil, fmt.Errorf("article title is empty")
	}
	if parsed.Link == "" {
		return nil, fmt.Errorf("article link is empty")
	}

	return parsed, nil
}

// parseRSSDate парсит дату из RSS формата в time.Time
// RSS использует RFC 2822 формат, например: "Mon, 06 Sep 2021 12:00:00 GMT"
func (p *Parser) parseRSSDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Список возможных форматов даты в RSS
	formats := []string{
		time.RFC1123Z,               // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,                // "02 Jan 06 15:04 -0700"
		time.RFC822,                 // "02 Jan 06 15:04 MST"
		"2006-01-02T15:04:05Z07:00", // ISO 8601
		"2006-01-02 15:04:05",       // Простой формат
		"2006-01-02",                // Только дата
	}

	// Пробуем каждый формат
	for _, format := range formats {
		if parsedTime, err := time.Parse(format, dateStr); err == nil {
			return parsedTime, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// ValidateRSSURL проверяет, является ли URL валидным RSS источником
func (p *Parser) ValidateRSSURL(url string) error {
	logger.Info("Validating RSS URL: %s", url)

	// Пробуем получить и парсить RSS ленту
	_, err := p.FetchAndParse(url)
	if err != nil {
		return fmt.Errorf("RSS URL validation failed: %w", err)
	}

	logger.Info("RSS URL is valid: %s", url)
	return nil
}
