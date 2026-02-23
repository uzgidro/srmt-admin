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

type AllStructuresGetter interface {
	GetAllStructures(ctx context.Context) ([]*salary.SalaryStructure, error)
}

func GetAllStructures(log *slog.Logger, svc AllStructuresGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetAllStructures"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		structures, err := svc.GetAllStructures(r.Context())
		if err != nil {
			log.Error("failed to get salary structures", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary structures"))
			return
		}

		render.JSON(w, r, structures)
	}
}
