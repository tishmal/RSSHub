FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static build (portable, no missing .so problems)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o rsshub .

# Small final image
FROM alpine:latest

RUN apk add --no-cache fish

WORKDIR /app
COPY --from=builder /app/rsshub  .

SHELL ["/usr/bin/fish", "-c"]

ENTRYPOINT ["./rsshub", "fetch"]
