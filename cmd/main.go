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
	"srmt-admin/internal/http-server/handlers/auth/sign-in"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/http-server/middleware/logger"
	startupadmin "srmt-admin/internal/lib/admin/startup-admin"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage/sqlite"
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
	log.Info("Logger start")

	storage, err := sqlite.New(cfg.StoragePath, cfg.MigrationsPath)
	if err != nil {
		log.Error("Error starting storage", sl.Err(err))
		os.Exit(1)
	}
	log.Info("Storage start")

	t, err := token.New(cfg.JwtConfig.Secret, cfg.JwtConfig.AccessTimeout, cfg.JwtConfig.RefreshTimeout)

	defer func() {
		if closeErr := StorageCloser.Close(storage); closeErr != nil {
			log.Error("Error closing storage", sl.Err(closeErr))
		}
	}()

	if err := startupadmin.EnsureAdminExists(context.Background(), log, storage); err != nil {
		log.Error("failed to ensure admin exists", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.New(log))
	r.Use(middleware.Recoverer)

	//r.Post("/auth/sign-up", sign_up.New(log, storage))
	r.Get("/auth/sign-in", sign_in.New(log, storage, t))

	r.Group(func(r chi.Router) {
		r.Use(mwauth.NewMiddleware(t))

		// Пример защищенного хендлера
		r.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
			isAdmin := mwauth.IsAdmin(r.Context())
			if !isAdmin {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		})
	})

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
