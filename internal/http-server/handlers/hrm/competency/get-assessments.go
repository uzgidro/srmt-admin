package competency

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/competency"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AssessmentAllGetter interface {
	GetAllAssessments(ctx context.Context, filters dto.AssessmentFilters) ([]*competency.AssessmentSession, error)
}

func GetAssessments(log *slog.Logger, svc AssessmentAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetAssessments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.AssessmentFilters

		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		result, err := svc.GetAllAssessments(r.Context(), filters)
		if err != nil {
			log.Error("failed to get assessments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve assessments"))
			return
		}

		render.JSON(w, r, result)
	}
}
