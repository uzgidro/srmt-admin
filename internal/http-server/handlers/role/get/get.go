package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/role"
)

type RoleGetter interface {
	GetAllRoles(ctx context.Context) ([]role.Model, error)
}

func New(log *slog.Logger, roleGetter RoleGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.role.get.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		roles, err := roleGetter.GetAllRoles(r.Context())
		if err != nil {
			log.Error("failed to get all roles", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve roles"))
			return
		}

		log.Info("successfully retrieved all roles", slog.Int("count", len(roles)))

		render.JSON(w, r, roles)
	}
}
