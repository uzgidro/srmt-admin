package invest_active_projects

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	invest_active_project "srmt-admin/internal/lib/model/invest-active-project"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investActiveProjectGetter interface {
	GetAllInvestActiveProjects(ctx context.Context) ([]*invest_active_project.Model, error)
}

func GetAll(log *slog.Logger, getter investActiveProjectGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.invest_active_projects.get-all"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		projects, err := getter.GetAllInvestActiveProjects(r.Context())
		if err != nil {
			log.Error("failed to get all active projects", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve active projects"))
			return
		}

		log.Info("successfully retrieved active projects", slog.Int("count", len(projects)))
		render.JSON(w, r, projects)
	}
}
