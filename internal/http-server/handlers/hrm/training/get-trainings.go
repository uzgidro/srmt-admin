package training

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/training"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type TrainingAllGetter interface {
	GetAllTrainings(ctx context.Context, filters dto.TrainingFilters) ([]*training.Training, error)
}

func GetTrainings(log *slog.Logger, svc TrainingAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetTrainings"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.TrainingFilters

		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("type"); v != "" {
			filters.Type = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		trainings, err := svc.GetAllTrainings(r.Context(), filters)
		if err != nil {
			log.Error("failed to get trainings", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve trainings"))
			return
		}

		render.JSON(w, r, trainings)
	}
}
