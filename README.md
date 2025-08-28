# Примеры использования RSSHub

## Базовое использование

### 1. Запуск с Docker Compose

```bash
# Клонируем проект
git clone <repository>
cd rsshub

# Запускаем сервисы
docker-compose up -d

# Проверяем статус
docker-compose ps
```

### 2. Добавление RSS лент

```bash
# Добавляем различные RSS ленты
./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
./rsshub add --name "hacker-news" --url "https://news.ycombinator.com/rss"
./rsshub add --name "bbc-world" --url "http://feeds.bbci.co.uk/news/world/rss.xml"
./rsshub add --name "the-verge" --url "https://www.theverge.com/rss/index.xml"
./rsshub add --name "ars-technica" --url "http://feeds.arstechnica.com/arstechnica/index"
```

### 3. Просмотр лент

```bash
# Показать все ленты
./rsshub list

# Показать только 3 последние ленты
./rsshub list --num 3
```

### 4. Запуск фонового агрегатора

В одном терминале:
```bash
# Запускаем фоновый процесс получения лент
./rsshub fetch
```

В другом терминале (для управления):
```bash
# Изменяем интервал на 2 минуты
./rsshub set-interval 2m

# Увеличиваем количество воркеров до 5
./rsshub set-workers 5
```

### 5. Просмотр статей

```bash
# Показать 5 последних статей из TechCrunch
./rsshub articles --feed-name "tech-crunch" --num 5

# Показать 3 статьи из Hacker News (по умолчанию)
./rsshub articles --feed-name "hacker-news"
```

### 6. Удаление лент

```bash
# Удалить ленту
./rsshub delete --name "tech-crunch"
```

## Расширенные сценарии

### Автоматический мониторинг новостей

1. Добавляем несколько источников новостей:
```bash
./rsshub add --name "bbc" --url "http://feeds.bbci.co.uk/news/world/rss.xml"
./rsshub add --name "cnn" --url "http://rss.cnn.com/rss/edition.rss"
./rsshub add --name "reuters" --url "https://www.reutersagency.com/feed/?best-topics=tech"
```

2. Запускаем агрегатор с частыми обновлениями:
```bash
./rsshub fetch
# В другом терминале:
./rsshub set-interval 1m
./rsshub set-workers 6
```

3. Периодически проверяем новости:
```bash
# Скрипт для проверки последних новостей
for feed in bbc cnn reuters; do
    echo "=== $feed ==="
    ./rsshub articles --feed-name "$feed" --num 3
    echo
done
```

### Мониторинг технологических блогов

```bash
# Добавляем технологические источники
./rsshub add --name "github-blog" --url "https://github.blog/feed/"
./rsshub add --name "stackoverflow-blog" --url "https://stackoverflow.blog/feed/"
./rsshub add --name "dev-to" --url "https://dev.to/feed"

# Запускаем с оптимальными настройками для блогов
./rsshub fetch
./rsshub set-interval 5m  # Блоги обновляются реже
./rsshub set-workers 3
```

## Troubleshooting

### Проблема: База данных недоступна
```bash
# Проверяем статус PostgreSQL
docker-compose ps postgres

# Смотрим логи
docker-compose logs postgres

# Перезапускаем базу данных
docker-compose restart postgres
```

### Проблема: RSS лента не парсится
```bash
# Проверяем URL вручную
curl -I "https://example.com/feed"

# Смотрим логи агрегатора
docker-compose logs rsshub
```

### Проблема: Слишком много дубликатов
```bash
# Уменьшаем интервал проверки
./rsshub set-interval 10m

# Или уменьшаем количество воркеров
./rsshub set-workers 2
```

## Полезные команды для разработки

```bash
# Сборка и запуск
make build
make run

# Форматирование кода
make format

# Проверка на race conditions
make check

# Подключение к базе данных
make db-connect

# Просмотр логов
make docker-logs

# Просмотр бд в консоли
make docker exec -it rsshub_postgres psql -U postgres -d rsshub
```