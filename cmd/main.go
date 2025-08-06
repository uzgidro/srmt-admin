package main

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/middleware/logger"
	"srmt-admin/internal/http-server/router"
	startupadmin "srmt-admin/internal/lib/admin/startup-admin"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage/driver/mongo"
	"srmt-admin/internal/storage/driver/postgres"
	mngRepo "srmt-admin/internal/storage/mongo"
	pgRepo "srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("logger start")

	pgDriver, err := postgres.New(cfg.StoragePath, cfg.MigrationsPath)
	if err != nil {
		log.Error("Error starting pgDriver", sl.Err(err))
		os.Exit(1)
	}
	log.Info("pgDriver start")

	repository := pgRepo.New(pgDriver)

	log.Info("repository start")

	mngClient, err := mongo.New(context.Background(), cfg.Mongo)
	if err != nil {
		log.Error("Error starting mongo", sl.Err(err))
		os.Exit(1)
	}

	mngRepository := mngRepo.New(mngClient)

	t, err := token.New(cfg.JwtConfig.Secret, cfg.JwtConfig.AccessTimeout, cfg.JwtConfig.RefreshTimeout)

	log.Info("token start")

	defer func() {
		if closeErr := StorageCloser.Close(repository); closeErr != nil {
			log.Error("Error closing storage", sl.Err(closeErr))
		}
		if closeErr := mngRepository.Close(context.Background()); closeErr != nil {
			log.Error("Error closing storage", sl.Err(closeErr))
		}
	}()

	if err := startupadmin.EnsureAdminExists(context.Background(), log, repository); err != nil {
		log.Error("failed to ensure admin exists", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.New(log))
	r.Use(middleware.Recoverer)

	router.SetupRoutes(r, log, t, repository, mngRepository)

	srv := &http.Server{
		Addr:         cfg.HttpServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HttpServer.Timeout,
		WriteTimeout: cfg.HttpServer.Timeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("Error starting http server", sl.Err(err))
	}

	log.Error("Server shutdown")
}

type StorageCloser interface {
	Close() error
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
