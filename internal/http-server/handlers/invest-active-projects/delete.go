package invest_active_projects

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investActiveProjectDeleter interface {
	DeleteInvestActiveProject(ctx context.Context, id int64) error
}

func Delete(log *slog.Logger, deleter investActiveProjectDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.invest_active_projects.delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		err = deleter.DeleteInvestActiveProject(r.Context(), id)
		if err != nil {
			if err == storage.ErrNotFound {
				log.Warn("active project not found for deletion", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Active project not found"))
				return
			}
			log.Error("failed to delete active project", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete active project"))
			return
		}

		log.Info("active project deleted successfully", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
