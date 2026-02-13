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

type CompetencyAllGetter interface {
	GetAllCompetencies(ctx context.Context, filters dto.CompetencyFilters) ([]*competency.Competency, error)
}

func GetCompetencies(log *slog.Logger, svc CompetencyAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetCompetencies"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.CompetencyFilters

		if v := q.Get("category"); v != "" {
			filters.Category = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		result, err := svc.GetAllCompetencies(r.Context(), filters)
		if err != nil {
			log.Error("failed to get competencies", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve competencies"))
			return
		}

		render.JSON(w, r, result)
	}
}
