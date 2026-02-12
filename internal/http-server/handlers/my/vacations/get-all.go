package vacations

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	vacationmodel "srmt-admin/internal/lib/model/hrm/vacation"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllGetter interface {
	GetAll(ctx context.Context, filters dto.VacationFilters) ([]*vacationmodel.Vacation, error)
}

func GetAll(log *slog.Logger, svc AllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.vacations.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		filters := dto.VacationFilters{
			EmployeeID: &claims.ContactID,
		}

		q := r.URL.Query()
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("vacation_type"); v != "" {
			filters.VacationType = &v
		}

		vacations, err := svc.GetAll(r.Context(), filters)
		if err != nil {
			log.Error("failed to get vacations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacations"))
			return
		}

		if vacations == nil {
			vacations = []*vacationmodel.Vacation{}
		}
		render.JSON(w, r, vacations)
	}
}
