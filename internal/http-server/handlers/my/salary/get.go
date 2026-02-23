package salary

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	salary "srmt-admin/internal/lib/model/hrm/salary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type SalaryGetter interface {
	GetAll(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error)
}

func Get(log *slog.Logger, svc SalaryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.salary.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		contactID := claims.ContactID
		filters := dto.SalaryFilters{EmployeeID: &contactID}

		salaries, err := svc.GetAll(r.Context(), filters)
		if err != nil {
			log.Error("failed to get salaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get salary data"))
			return
		}

		render.JSON(w, r, salaries)
	}
}
