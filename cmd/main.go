package main

import (
	"log/slog"
	"os"
	"srmt-admin/internal/config"
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

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("Error starting storage", sl.Err(err))
		os.Exit(1)
	}
	log.Info("Storage start")

	defer func() {
		if closeErr := storage.DB.Close(); closeErr != nil {
			log.Error("Error closing storage", sl.Err(closeErr))
		}
	}()
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
