package performance

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/performance"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type RatingGetter interface {
	GetAllRatings(ctx context.Context) ([]*performance.EmployeeRating, error)
}

func GetRatings(log *slog.Logger, svc RatingGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetRatings"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		result, err := svc.GetAllRatings(r.Context())
		if err != nil {
			log.Error("failed to get ratings", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve ratings"))
			return
		}

		render.JSON(w, r, result)
	}
}

type EmployeeRatingGetter interface {
	GetEmployeeRating(ctx context.Context, employeeID int64) (*performance.EmployeeRating, error)
}

func GetEmployeeRating(log *slog.Logger, svc EmployeeRatingGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetEmployeeRating"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		result, err := svc.GetEmployeeRating(r.Context(), id)
		if err != nil {
			log.Error("failed to get employee rating", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee rating"))
			return
		}

		render.JSON(w, r, result)
	}
}
