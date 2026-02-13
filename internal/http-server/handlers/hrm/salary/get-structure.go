package salary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	salary "srmt-admin/internal/lib/model/hrm/salary"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type StructureGetter interface {
	GetStructure(ctx context.Context, employeeID int64) ([]*salary.SalaryStructure, error)
}

func GetStructure(log *slog.Logger, svc StructureGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetStructure"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		employeeID, err := strconv.ParseInt(chi.URLParam(r, "employeeId"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid employee ID"))
			return
		}

		structures, err := svc.GetStructure(r.Context(), employeeID)
		if err != nil {
			log.Error("failed to get salary structure", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary structure"))
			return
		}

		render.JSON(w, r, structures)
	}
}
