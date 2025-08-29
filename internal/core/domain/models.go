package domain

import (
	"time"

	"rsshub/internal/platform/utils"
)

// Feed представляет RSS ленту в базе данных
type Feed struct {
	ID        utils.UUID `json:"id"`         // Уникальный идентификатор
	CreatedAt time.Time  `json:"created_at"` // Время создания записи
	UpdatedAt time.Time  `json:"updated_at"` // Время последнего обновления
	Name      string     `json:"name"`       // Человекочитаемое имя ленты
	URL       string     `json:"url"`        // URL для получения RSS данных
}

// Article представляет статью в базе данных
type Article struct {
	ID          utils.UUID `json:"id"`           // Уникальный идентификатор
	CreatedAt   time.Time  `json:"created_at"`   // Время создания записи
	UpdatedAt   time.Time  `json:"updated_at"`   // Время последнего обновления
	Title       string     `json:"title"`        // Заголовок статьи
	Link        string     `json:"link"`         // URL статьи
	PublishedAt time.Time  `json:"published_at"` // Дата публикации из RSS
	Description string     `json:"description"`  // Описание статьи
	FeedID      utils.UUID `json:"feed_id"`      // ID ленты, к которой принадлежит статья
}

// RSSFeed представляет структуру RSS XML документа
// Используется для парсинга XML ответов от RSS серверов
type RSSFeed struct {
	Channel RSSChannel `xml:"channel"` // Основной канал с информацией о ленте
}

// RSSChannel содержит метаданные канала и список элементов
type RSSChannel struct {
	Title       string    `xml:"title"`       // Название канала
	Link        string    `xml:"link"`        // Ссылка на сайт
	Description string    `xml:"description"` // Описание канала
	Items       []RSSItem `xml:"item"`        // Список статей/элементов
}

// RSSItem представляет отдельную статью в RSS ленте
type RSSItem struct {
	Title       string `xml:"title"`       // Заголовок статьи
	Link        string `xml:"link"`        // Ссылка на статью
	Description string `xml:"description"` // Описание/краткое содержание
	PubDate     string `xml:"pubDate"`     // Дата публикации в RSS формате
}

// ParsedRSSFeed представляет распарсенную RSS ленту с преобразованными данными
type ParsedRSSFeed struct {
	Title       string          // Название канала
	Link        string          // Ссылка на сайт
	Description string          // Описание канала
	Items       []ParsedRSSItem // Список обработанных статей
}

// ParsedRSSItem представляет обработанную статью с корректно распарсенной датой
type ParsedRSSItem struct {
	Title       string    // Заголовок статьи
	Link        string    // Ссылка на статью
	Description string    // Описание статьи
	PublishedAt time.Time // Дата публикации как time.Time
}
