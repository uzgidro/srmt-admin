package training

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/training"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type EmployeeTrainingsGetter interface {
	GetEmployeeTrainings(ctx context.Context, employeeID int64) ([]*training.Training, error)
}

func GetEmployeeTrainings(log *slog.Logger, svc EmployeeTrainingsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetEmployeeTrainings"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		employeeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		trainings, err := svc.GetEmployeeTrainings(r.Context(), employeeID)
		if err != nil {
			log.Error("failed to get employee trainings", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee trainings"))
			return
		}

		render.JSON(w, r, trainings)
	}
}
