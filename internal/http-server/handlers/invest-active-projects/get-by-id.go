package invest_active_projects

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	invest_active_project "srmt-admin/internal/lib/model/invest-active-project"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investActiveProjectByIDGetter interface {
	GetInvestActiveProjectByID(ctx context.Context, id int64) (*invest_active_project.Model, error)
}

func GetByID(log *slog.Logger, getter investActiveProjectByIDGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.invest_active_projects.get-by-id"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		project, err := getter.GetInvestActiveProjectByID(r.Context(), id)
		if err != nil {
			if err == storage.ErrNotFound {
				log.Warn("active project not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Active project not found"))
				return
			}
			log.Error("failed to get active project", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get active project"))
			return
		}

		log.Info("successfully retrieved active project", slog.Int64("id", id))
		render.JSON(w, r, project)
	}
}
