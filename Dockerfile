# Используем официальный Go образ
FROM golang:1.24-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o rsshub .

# Запускаем приложение
ENTRYPOINT ["./rsshub", "fetch"]
