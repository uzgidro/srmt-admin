package salary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	salary "srmt-admin/internal/lib/model/hrm/salary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllDeductionsGetter interface {
	GetAllDeductions(ctx context.Context) ([]*salary.Deduction, error)
}

func GetAllDeductions(log *slog.Logger, svc AllDeductionsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetAllDeductions"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		deductions, err := svc.GetAllDeductions(r.Context())
		if err != nil {
			log.Error("failed to get deductions", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve deductions"))
			return
		}

		render.JSON(w, r, deductions)
	}
}
