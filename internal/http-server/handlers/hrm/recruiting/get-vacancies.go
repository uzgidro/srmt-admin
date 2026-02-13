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

type VacancyAllGetter interface {
	GetAllVacancies(ctx context.Context, filters dto.VacancyFilters) ([]*recruiting.Vacancy, error)
}

func GetVacancies(log *slog.Logger, svc VacancyAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetVacancies"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.VacancyFilters

		if v := q.Get("department_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.DepartmentID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("priority"); v != "" {
			filters.Priority = &v
		}
		if v := q.Get("employment_type"); v != "" {
			filters.EmploymentType = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		vacancies, err := svc.GetAllVacancies(r.Context(), filters)
		if err != nil {
			log.Error("failed to get vacancies", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacancies"))
			return
		}

		render.JSON(w, r, vacancies)
	}
}
