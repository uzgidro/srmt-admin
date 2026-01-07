package invest_active_projects

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investActiveProjectAdder interface {
	AddInvestActiveProject(ctx context.Context, req dto.AddInvestActiveProjectRequest) (int64, error)
}

func Add(log *slog.Logger, adder investActiveProjectAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.invest_active_projects.add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req dto.AddInvestActiveProjectRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Warn("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request body"))
			return
		}

		// Basic validation
		if req.Category == "" || req.ProjectName == "" {
			log.Warn("missing required fields")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Category and ProjectName are required"))
			return
		}

		id, err := adder.AddInvestActiveProject(r.Context(), req)
		if err != nil {
			log.Error("failed to add active project", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add active project"))
			return
		}

		log.Info("active project added successfully", slog.Int64("id", id))
		render.JSON(w, r, map[string]interface{}{
			"status": "success",
			"id":     id,
		})
	}
}
