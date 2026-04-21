# --- Builder ---
FROM golang:1.26.2-alpine AS builder

RUN apk add --no-cache git
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/google/wire/cmd/wire@latest

COPY . .
RUN cd cmd && wire
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o srmt-admin ./cmd

# --- Final ---
FROM alpine:3.23

# Runtime deps: ca-certificates, tzdata, libreoffice (Excel→PDF),
# fonts with Cyrillic support, libpq
RUN apk upgrade --no-cache && \
    apk --no-cache add ca-certificates tzdata libreoffice \
    font-dejavu font-liberation font-noto \
    msttcorefonts-installer fontconfig \
    musl-locales musl-locales-lang && \
    apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main libpq && \
    update-ms-fonts && \
    fc-cache -f

ENV LANG=ru_RU.UTF-8
ENV LC_ALL=ru_RU.UTF-8

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app
COPY --from=builder /build/srmt-admin .
COPY --from=builder /build/migrations ./migrations
RUN mkdir -p /app/config && chown -R appuser:appuser /app

USER appuser
EXPOSE 9010

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:9010/ping || exit 1

ENV CONFIG_PATH=/app/config/prod.yaml
CMD ["./srmt-admin"]