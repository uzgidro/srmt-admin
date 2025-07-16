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
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	usersAdd "srmt-admin/internal/http-server/handlers/users/add"
	assignRole "srmt-admin/internal/http-server/handlers/users/assign-role"
	usersEdit "srmt-admin/internal/http-server/handlers/users/edit"
	revokeRole "srmt-admin/internal/http-server/handlers/users/revoke-role"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/http-server/middleware/logger"
	startupadmin "srmt-admin/internal/lib/admin/startup-admin"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/storage/driver/postgres"
	"srmt-admin/internal/storage/driver/sqlite"
	"srmt-admin/internal/storage/repo"
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

	driver, err := setupDriver(cfg.Env, cfg.StoragePath, cfg.MigrationsPath)
	if err != nil {
		log.Error("Error starting storage", sl.Err(err))
		os.Exit(1)
	}
	log.Info("driver start")

	repository := repo.New(driver)

	t, err := token.New(cfg.JwtConfig.Secret, cfg.JwtConfig.AccessTimeout, cfg.JwtConfig.RefreshTimeout)

	defer func() {
		if closeErr := StorageCloser.Close(repository); closeErr != nil {
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

	r.Get("/auth/sign-in", sign_in.New(log, repository, t))

	// Admin endpoints
	r.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(t))
		r.Use(mwauth.AdminOnly)

		// Roles
		r.Post("/roles", roleAdd.New(log, repository))
		r.Patch("/roles/{id}", roleEdit.New(log, repository))
		r.Delete("/roles/{id}", roleDelete.New(log, repository))

		// Users
		r.Post("/users", usersAdd.New(log, repository))
		r.Patch("/users", usersEdit.New(log, repository))
		r.Post("/users/{userID}/roles", assignRole.New(log, repository))
		r.Delete("/users/{userID}/roles/{roleID}", revokeRole.New(log, repository))
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

func setupDriver(env, storagePath, migrationsPath string) (*storage.Driver, error) {
	var driver *storage.Driver
	var err error

	switch env {
	case envLocal:
		driver, err = sqlite.New(storagePath, migrationsPath)
	case envDev:
		driver, err = postgres.New(storagePath, migrationsPath)
	}
	return driver, err
}
