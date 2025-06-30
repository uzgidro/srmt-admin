package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/middleware/logger"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage/sqlite"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("Logger start")

	storage, err := sqlite.New(cfg.StoragePath, cfg.MigrationsPath)
	if err != nil {
		log.Error("Error starting storage", sl.Err(err))
		os.Exit(1)
	}
	log.Info("Storage start")

	defer func() {
		if closeErr := storage.Close(); closeErr != nil {
			log.Error("Error closing storage", sl.Err(closeErr))
		}
	}()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.New(log))
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(cfg.HttpServer.IdleTimeout))

	http.ListenAndServe(cfg.HttpServer.Address, r)
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
