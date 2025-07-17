package router

import (
	"github.com/go-chi/chi/v5"
	"log/slog"
	sign_in "srmt-admin/internal/http-server/handlers/auth/sign-in"
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	usersAdd "srmt-admin/internal/http-server/handlers/users/add"
	assignRole "srmt-admin/internal/http-server/handlers/users/assign-role"
	usersEdit "srmt-admin/internal/http-server/handlers/users/edit"
	revokeRole "srmt-admin/internal/http-server/handlers/users/revoke-role"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
)

func SetupRoutes(router *chi.Mux, log *slog.Logger, token *token.Token, repository *repo.Repo) {
	router.Post("/auth/sign-in", sign_in.New(log, repository, token))

	// Admin endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(token))
		r.Use(mwauth.AdminOnly)

		// Roles
		r.Post("/roles", roleAdd.New(log, repository))
		r.Patch("/roles/{id}", roleEdit.New(log, repository))
		r.Delete("/roles/{id}", roleDelete.New(log, repository))

		// Users
		r.Post("/users", usersAdd.New(log, repository))
		r.Patch("/users/{userID}", usersEdit.New(log, repository))
		r.Post("/users/{userID}/roles", assignRole.New(log, repository))
		r.Delete("/users/{userID}/roles/{roleID}", revokeRole.New(log, repository))
	})
}
