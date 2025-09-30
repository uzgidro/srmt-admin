package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/handlers/auth/refresh"
	signIn "srmt-admin/internal/http-server/handlers/auth/sign-in"
	"srmt-admin/internal/http-server/handlers/data/analytics"
	dataSet "srmt-admin/internal/http-server/handlers/data/set"
	"srmt-admin/internal/http-server/handlers/file/category"
	"srmt-admin/internal/http-server/handlers/file/upload"
	setIndicator "srmt-admin/internal/http-server/handlers/indicators/set"
	resAdd "srmt-admin/internal/http-server/handlers/reservoirs/add"
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	callbackModsnow "srmt-admin/internal/http-server/handlers/sc/callback/modsnow"
	callbackStock "srmt-admin/internal/http-server/handlers/sc/callback/stock"
	modsnowImg "srmt-admin/internal/http-server/handlers/sc/modsnow/img"
	"srmt-admin/internal/http-server/handlers/sc/modsnow/table"
	"srmt-admin/internal/http-server/handlers/sc/stock"
	usersAdd "srmt-admin/internal/http-server/handlers/users/add"
	assignRole "srmt-admin/internal/http-server/handlers/users/assign-role"
	usersEdit "srmt-admin/internal/http-server/handlers/users/edit"
	revokeRole "srmt-admin/internal/http-server/handlers/users/revoke-role"
	weatherProxy "srmt-admin/internal/http-server/handlers/weather/proxy"
	mwapikey "srmt-admin/internal/http-server/middleware/api-key"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage/minio"
	"srmt-admin/internal/storage/mongo"
	"srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
)

func SetupRoutes(router *chi.Mux, log *slog.Logger, token *token.Token, pg *repo.Repo, mng *mongo.Repo, minioClient *minio.Repo, cfg config.Config) {
	router.Post("/auth/sign-in", signIn.New(log, pg, token))
	router.Post("/auth/refresh", refresh.New(log, pg, token))

	router.Route("/api/v3", func(r chi.Router) {
		r.Get("/modsnow", table.Get(log, mng))
		r.Get("/stock", stock.Get(log, mng))
		r.Get("/modsnow/cover", modsnowImg.Get(log, minioClient, "modsnow-cover"))
		r.Get("/modsnow/dynamics", modsnowImg.Get(log, minioClient, "modsnow-dynamics"))

		r.Get("/analytics", analytics.New(log, pg))

		r.Route("/weather", func(r chi.Router) {
			httpClient := &http.Client{}
			weatherCfg := cfg.Weather

			r.Get("/weather", weatherProxy.New(log, httpClient, weatherCfg.BaseURL, weatherCfg.APIKey, "/weather"))
			r.Get("/forecast", weatherProxy.New(log, httpClient, weatherCfg.BaseURL, weatherCfg.APIKey, "/forecast"))
		})

	})

	// Service endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwapikey.RequireAPIKey(cfg.ApiKey))

		r.Post("/sc/stock", callbackStock.New(log, mng))
		r.Post("/sc/modsnow", callbackModsnow.New(log, mng))
		r.Post("/data/{id}", dataSet.New(log, pg))
	})

	// Admin endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(token))
		r.Use(mwauth.AdminOnly)

		// Roles
		r.Post("/roles", roleAdd.New(log, pg))
		r.Patch("/roles/{id}", roleEdit.New(log, pg))
		r.Delete("/roles/{id}", roleDelete.New(log, pg))

		// Users
		r.Post("/users", usersAdd.New(log, pg))
		r.Patch("/users/{userID}", usersEdit.New(log, pg))
		r.Post("/users/{userID}/roles", assignRole.New(log, pg))
		r.Delete("/users/{userID}/roles/{roleID}", revokeRole.New(log, pg))

		// File category
		r.Post("/files/categories", category.New(log, pg))
	})

	// SC endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(token))
		r.Use(mwauth.RequireAnyRole("admin", "sc"))

		// Indicator
		r.Put("/indicators/{resID}", setIndicator.New(log, pg))

		// Upload
		r.Post("/upload/stock", stock.Upload(log, &http.Client{}, cfg.Upload.Stock))
		r.Post("/upload/modsnow", table.Upload(log, &http.Client{}, cfg.Upload.Modsnow))
		r.Post("/upload/archive", modsnowImg.Upload(log, &http.Client{}, cfg.Upload.Archive))
		r.Post("/upload/file", upload.New(log, minioClient, pg))

		// Reservoirs
		r.Post("/reservoirs", resAdd.New(log, pg))
	})
}
