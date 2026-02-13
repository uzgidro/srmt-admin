package orgstructure

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/orgstructure"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllEmployeeGetter interface {
	GetAllEmployees(ctx context.Context) ([]*orgstructure.OrgEmployee, error)
}

func GetEmployees(log *slog.Logger, svc AllEmployeeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.orgstructure.GetEmployees"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		employees, err := svc.GetAllEmployees(r.Context())
		if err != nil {
			log.Error("failed to get employees", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employees"))
			return
		}

		render.JSON(w, r, employees)
	}
}
