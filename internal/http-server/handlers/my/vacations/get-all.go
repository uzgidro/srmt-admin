package vacations

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/profile"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllGetter interface {
	GetMyVacations(ctx context.Context, employeeID int64, status *string, vacationType *string) ([]*profile.MyVacation, error)
}

func GetAll(log *slog.Logger, repo AllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.vacations.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		q := r.URL.Query()
		var status, vacationType *string
		if v := q.Get("status"); v != "" {
			status = &v
		}
		if v := q.Get("type"); v != "" {
			vacationType = &v
		}

		vacations, err := repo.GetMyVacations(r.Context(), claims.ContactID, status, vacationType)
		if err != nil {
			log.Error("failed to get vacations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacations"))
			return
		}

		if vacations == nil {
			vacations = []*profile.MyVacation{}
		}
		render.JSON(w, r, vacations)
	}
}
