package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	startupadmin "srmt-admin/internal/lib/admin/startup-admin"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/excel/templates"
)

func main() {
	// Initialize app with Wire
	app, cleanup, err := InitializeApp()
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	log := app.Logger
	log.Info("application initialized")
	log.Info("timezone configured", "timezone", app.Config.Timezone, "location", app.Location.String())

	// Echo the override-path setting at startup so the operator can sanity-
	// check that cleanenv actually picked up the value they wrote (a typo
	// in the YAML key — template_path vs template_override_path — would be
	// silently ignored, leaving the path empty here).
	overrideFiles := templates.CountOverrideFiles(app.Config.TemplateOverridePath)
	log.Info("excel templates configured",
		"override_path", app.Config.TemplateOverridePath,
		"override_files_found", overrideFiles,
		"embedded_files", len(templates.AllNames()))
	// Loud warn when the path is set but useless — most common cause is a
	// typo in the directory path. Without this dev would silently keep
	// getting embed copies and wonder why edits are ignored.
	if app.Config.TemplateOverridePath != "" && overrideFiles == 0 {
		log.Warn("template override path set but no embedded template files found there; using embed-only",
			"path", app.Config.TemplateOverridePath)
	}

	// Ensure admin user exists
	if err := startupadmin.EnsureAdminExists(context.Background(), log, app.PgRepo); err != nil {
		log.Error("failed to ensure admin exists", "error", err)
		os.Exit(1)
	}

	// Start day rotation scheduler
	rotationCtx, rotationCancel := context.WithCancel(context.Background())
	defer rotationCancel()
	if app.DayRotationService != nil {
		go app.DayRotationService.StartScheduler(rotationCtx)
	}

	// Start HTTP server with graceful shutdown
	log.Info("starting http server", "address", app.Config.HttpServer.Address)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", sl.Err(err))
		}
	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.Server.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", sl.Err(err))
		}
	}

	log.Info("server stopped")
}
