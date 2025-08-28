-- Создание таблицы для RSS лент
-- UUID расширение для генерации уникальных идентификаторов
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL UNIQUE, -- Уникальное человекочитаемое имя ленты
    url TEXT NOT NULL           -- URL для получения RSS контента
);

-- Индекс для быстрого поиска по имени
CREATE INDEX idx_feeds_name ON feeds(name);

-- Индекс для сортировки по времени обновления (для выбора устаревших лент)
CREATE INDEX idx_feeds_updated_at ON feeds(updated_at);