# --- Этап 1: Сборка (Builder) ---
# Используем полный образ Go для компиляции приложения
FROM golang:1.24.4-alpine AS builder

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git make

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем файлы зависимостей для кэширования слоев
COPY go.mod go.sum ./
# Загружаем зависимости
RUN go mod download

# Копируем весь исходный код
COPY . .

# ВАЖНО: Генерируем Wire dependency injection код
# Это необходимо перед сборкой, так как Wire создает wire_gen.go
RUN cd cmd && go run github.com/google/wire/cmd/wire

# Собираем приложение в статически слинкованный бинарник
# CGO_ENABLED=0 - отключает CGO, что важно для Alpine
# -ldflags="-s -w" - уменьшает размер бинарника, удаляя отладочную информацию
# ВАЖНО: Собираем весь пакет ./cmd, а не только main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o srmt-admin \
    ./cmd


# --- Этап 2: Финальный образ (Final) ---
# Используем минимальный базовый образ, который не содержит ничего лишнего
FROM alpine:latest

# Устанавливаем необходимые runtime зависимости
# ca-certificates - для HTTPS соединений
# tzdata - для поддержки временных зон
RUN apk --no-cache add ca-certificates tzdata

# Создаем непривилегированного пользователя для безопасности
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем ТОЛЬКО скомпилированный бинарник из этапа сборки
COPY --from=builder /build/srmt-admin .

# Копируем директорию с миграциями (необходимы для работы)
COPY --from=builder /build/migrations ./migrations

# Копируем директорию с шаблонами (необходимы для Excel экспорта)
COPY --from=builder /build/template ./template

# Создаем директорию для конфигов (будут монтироваться через volume)
RUN mkdir -p /app/config && chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Указываем порт (можно переопределить через переменные окружения)
EXPOSE 9010

# Health check для проверки работоспособности приложения
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9010/api/v3/analytics || exit 1

# Переменная окружения для пути к конфигу (переопределяется через docker-compose)
ENV CONFIG_PATH=/app/config/prod.yaml

# Команда для запуска приложения
CMD ["./srmt-admin"]