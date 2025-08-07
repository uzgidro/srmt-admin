package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	signIn "srmt-admin/internal/http-server/handlers/auth/sign-in"
	dataSet "srmt-admin/internal/http-server/handlers/data/set"
	setIndicator "srmt-admin/internal/http-server/handlers/indicators/set"
	resAdd "srmt-admin/internal/http-server/handlers/reservoirs/add"
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	"srmt-admin/internal/http-server/handlers/sc/archive"
	callbackModsnow "srmt-admin/internal/http-server/handlers/sc/callback/modsnow"
	callbackStock "srmt-admin/internal/http-server/handlers/sc/callback/stock"
	"srmt-admin/internal/http-server/handlers/sc/modsnow"
	"srmt-admin/internal/http-server/handlers/sc/stock"
	usersAdd "srmt-admin/internal/http-server/handlers/users/add"
	assignRole "srmt-admin/internal/http-server/handlers/users/assign-role"
	usersEdit "srmt-admin/internal/http-server/handlers/users/edit"
	revokeRole "srmt-admin/internal/http-server/handlers/users/revoke-role"
	mwapikey "srmt-admin/internal/http-server/middleware/api-key"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage/mongo"
	"srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
)

func SetupRoutes(router *chi.Mux, log *slog.Logger, token *token.Token, pg *repo.Repo, mng *mongo.Repo, apiKey string) {
	router.Post("/auth/sign-in", signIn.New(log, pg, token))

	router.Post("/data/{id}", dataSet.New(log, pg))

	// Parser callback endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwapikey.RequireAPIKey(apiKey))

		r.Post("/sc/stock", callbackStock.New(log, mng))
		r.Post("/sc/modsnow", callbackModsnow.New(log, mng))
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

	})

	// SC endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(token))
		r.Use(mwauth.RequireAnyRole("admin", "sc"))

		// Indicator
		r.Put("/indicators/{resID}", setIndicator.New(log, pg))

		// Upload
		r.Post("/upload/stock", stock.New(log, &http.Client{}))
		r.Post("/upload/modsnow", modsnow.New(log, &http.Client{}))
		r.Post("/upload/archive", archive.New(log, &http.Client{}))

		// Reservoirs
		r.Post("/reservoirs", resAdd.New(log, pg))
	})
}
