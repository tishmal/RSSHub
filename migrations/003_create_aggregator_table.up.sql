-- Создание таблицы для настройки агрегатора
CREATE TABLE aggregator (
id SERIAL PRIMARY KEY,
key TEXT UNIQUE NOT NULL,
value TEXT NOT NULL);

-- Индекс
CREATE INDEX idx_aggregator_id ON aggregator(id);