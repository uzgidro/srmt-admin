package invest_active_projects

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investActiveProjectEditor interface {
	EditInvestActiveProject(ctx context.Context, id int64, req dto.EditInvestActiveProjectRequest) error
}

func Edit(log *slog.Logger, editor investActiveProjectEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.invest_active_projects.edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		var req dto.EditInvestActiveProjectRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Warn("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request body"))
			return
		}

		err = editor.EditInvestActiveProject(r.Context(), id, req)
		if err != nil {
			if err == storage.ErrNotFound {
				log.Warn("active project not found for update", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Active project not found"))
				return
			}
			log.Error("failed to edit active project", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to edit active project"))
			return
		}

		log.Info("active project updated successfully", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
