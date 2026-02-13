package recruiting

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	recruiting "srmt-admin/internal/lib/model/hrm/recruiting"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type InterviewAllGetter interface {
	GetAllInterviews(ctx context.Context, filters dto.InterviewFilters) ([]*recruiting.Interview, error)
}

func GetInterviews(log *slog.Logger, svc InterviewAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetInterviews"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.InterviewFilters

		if v := q.Get("candidate_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.CandidateID = &val
		}
		if v := q.Get("vacancy_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.VacancyID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("type"); v != "" {
			filters.Type = &v
		}

		interviews, err := svc.GetAllInterviews(r.Context(), filters)
		if err != nil {
			log.Error("failed to get interviews", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve interviews"))
			return
		}

		render.JSON(w, r, interviews)
	}
}
