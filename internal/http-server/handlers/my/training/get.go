package training

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	training "srmt-admin/internal/lib/model/hrm/training"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type TrainingGetter interface {
	GetEmployeeTrainings(ctx context.Context, employeeID int64) ([]*training.Training, error)
}

func Get(log *slog.Logger, svc TrainingGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.training.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		trainings, err := svc.GetEmployeeTrainings(r.Context(), claims.ContactID)
		if err != nil {
			log.Error("failed to get trainings", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get training data"))
			return
		}

		if trainings == nil {
			trainings = []*training.Training{}
		}

		render.JSON(w, r, trainings)
	}
}
