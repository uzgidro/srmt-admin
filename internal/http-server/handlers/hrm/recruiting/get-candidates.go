package recruiting

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/recruiting"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CandidateAllGetter interface {
	GetAllCandidates(ctx context.Context, filters dto.CandidateFilters) ([]*recruiting.CandidateListItem, error)
}

func GetCandidates(log *slog.Logger, svc CandidateAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetCandidates"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.CandidateFilters

		if v := q.Get("vacancy_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.VacancyID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("stage"); v != "" {
			filters.Stage = &v
		}
		if v := q.Get("source"); v != "" {
			filters.Source = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		candidates, err := svc.GetAllCandidates(r.Context(), filters)
		if err != nil {
			log.Error("failed to get candidates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve candidates"))
			return
		}

		render.JSON(w, r, candidates)
	}
}
