package competency

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/competency"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type EmployeeAssessmentGetter interface {
	GetEmployeeAssessments(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error)
}

func GetEmployeeAssessments(log *slog.Logger, svc EmployeeAssessmentGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetEmployeeAssessments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		result, err := svc.GetEmployeeAssessments(r.Context(), id)
		if err != nil {
			log.Error("failed to get employee assessments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee assessments"))
			return
		}

		render.JSON(w, r, result)
	}
}
