-- Создание таблицы для статей
CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    title TEXT NOT NULL,                    -- Заголовок статьи
    link TEXT NOT NULL,                     -- URL статьи (должен быть уникальным)
    published_at TIMESTAMP,                 -- Дата публикации из RSS
    description TEXT,                       -- Описание/краткое содержание
    feed_id UUID NOT NULL,                  -- Связь с таблицей feeds
    
    -- Внешний ключ на таблицу feeds
    FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
    
    -- Уникальность по URL чтобы избежать дубликатов
    UNIQUE(link)
);

-- Индекс для быстрого поиска статей по ленте
CREATE INDEX idx_articles_feed_id ON articles(feed_id);

-- Индекс для сортировки по дате публикации
CREATE INDEX idx_articles_published_at ON articles(published_at DESC);

-- Композитный индекс для выборки последних статей конкретной ленты
CREATE INDEX idx_articles_feed_published ON articles(feed_id, published_at DESC);