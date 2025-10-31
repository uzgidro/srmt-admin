package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/handlers/auth/me"
	"srmt-admin/internal/http-server/handlers/auth/refresh"
	signIn "srmt-admin/internal/http-server/handlers/auth/sign-in"
	signOut "srmt-admin/internal/http-server/handlers/auth/sign-out"
	"srmt-admin/internal/http-server/handlers/data/analytics"
	dataSet "srmt-admin/internal/http-server/handlers/data/set"
	catAdd "srmt-admin/internal/http-server/handlers/file/category/add"
	catGet "srmt-admin/internal/http-server/handlers/file/category/list"
	fileDelete "srmt-admin/internal/http-server/handlers/file/delete"
	"srmt-admin/internal/http-server/handlers/file/download"
	"srmt-admin/internal/http-server/handlers/file/latest"
	"srmt-admin/internal/http-server/handlers/file/upload"
	setIndicator "srmt-admin/internal/http-server/handlers/indicators/set"
	orgAdd "srmt-admin/internal/http-server/handlers/organizations/add"
	orgDelete "srmt-admin/internal/http-server/handlers/organizations/delete"
	orgPatch "srmt-admin/internal/http-server/handlers/organizations/edit"
	orgGet "srmt-admin/internal/http-server/handlers/organizations/get"
	positionsAdd "srmt-admin/internal/http-server/handlers/positions/add"
	positionsDelete "srmt-admin/internal/http-server/handlers/positions/delete"
	positionsGet "srmt-admin/internal/http-server/handlers/positions/get"
	positionsPatch "srmt-admin/internal/http-server/handlers/positions/patch"
	resAdd "srmt-admin/internal/http-server/handlers/reservoirs/add"
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	roleGet "srmt-admin/internal/http-server/handlers/role/get"
	callbackModsnow "srmt-admin/internal/http-server/handlers/sc/callback/modsnow"
	callbackStock "srmt-admin/internal/http-server/handlers/sc/callback/stock"
	modsnowImg "srmt-admin/internal/http-server/handlers/sc/modsnow/img"
	"srmt-admin/internal/http-server/handlers/sc/modsnow/table"
	"srmt-admin/internal/http-server/handlers/sc/stock"
	"srmt-admin/internal/http-server/handlers/telegram/gidro/test"
	usersAdd "srmt-admin/internal/http-server/handlers/users/add"
	assignRole "srmt-admin/internal/http-server/handlers/users/assign-role"
	usersEdit "srmt-admin/internal/http-server/handlers/users/edit"
	usersGet "srmt-admin/internal/http-server/handlers/users/get"
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
	router.Post("/auth/sign-out", signOut.New(log))

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

		r.Get("/telegram/gidro/test", test.New(log, mng))
	})

	// Service endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwapikey.RequireAPIKey(cfg.ApiKey))

		r.Post("/sc/stock", callbackStock.New(log, mng))
		r.Post("/sc/modsnow", callbackModsnow.New(log, mng))
		r.Post("/data/{id}", dataSet.New(log, pg))
	})

	// Token required routes
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(token))

		r.Get("/auth/me", me.New(log))

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(mwauth.AdminOnly)

			// Roles
			r.Get("/roles", roleGet.New(log, pg))
			r.Post("/roles", roleAdd.New(log, pg))
			r.Patch("/roles/{id}", roleEdit.New(log, pg))
			r.Delete("/roles/{id}", roleDelete.New(log, pg))

			// Positions
			r.Get("/positions", positionsGet.New(log, pg))
			r.Post("/positions", positionsAdd.New(log, pg))
			r.Patch("/positions/{id}", positionsPatch.New(log, pg))
			r.Delete("/positions/{id}", positionsDelete.New(log, pg))

			// Organizations
			r.Get("/organizations", orgGet.New(log, pg))
			r.Post("/organizations", orgAdd.New(log, pg))
			r.Patch("/organizations/{id}", orgPatch.New(log, pg))
			r.Delete("/organizations/{id}", orgDelete.New(log, pg))

			// Users
			r.Get("/users", usersGet.New(log, pg))
			r.Post("/users", usersAdd.New(log, pg))
			r.Patch("/users/{userID}", usersEdit.New(log, pg))
			r.Post("/users/{userID}/roles", assignRole.New(log, pg))
			r.Delete("/users/{userID}/roles/{roleID}", revokeRole.New(log, pg))
		})

		// SC endpoints
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("sc"))

			// Indicator
			r.Put("/indicators/{resID}", setIndicator.New(log, pg))

			// Upload
			r.Post("/upload/stock", stock.Upload(log, &http.Client{}, cfg.Upload.Stock))
			r.Post("/upload/modsnow", table.Upload(log, &http.Client{}, cfg.Upload.Modsnow))
			r.Post("/upload/archive", modsnowImg.Upload(log, &http.Client{}, cfg.Upload.Archive))
			r.Post("/upload/files", upload.New(log, minioClient, pg))

			// Reservoirs
			r.Post("/reservoirs", resAdd.New(log, pg))

			// File category
			r.Get("/files/categories", catGet.New(log, pg))
			r.Post("/files/categories", catAdd.New(log, pg))

			// Delete
			r.Delete("/files/{fileID}", fileDelete.New(log, pg, minioClient))
		})

		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("sc", "rais"))

			r.Get("/files/latest", latest.New(log, pg, minioClient))
			r.Get("/files/{fileID}/download", download.New(log, pg, minioClient))
		})

	})
}
