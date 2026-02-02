package departments

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/department"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// DepartmentGetter defines the interface for getting departments by organization ID
type DepartmentGetter interface {
	GetDepartmentsByOrgID(ctx context.Context, orgID int64) ([]*department.Model, error)
}

// New creates a handler for GET /ges/{id}/departments
func New(log *slog.Logger, getter DepartmentGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.departments.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		orgID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		log = log.With(slog.Int64("organization_id", orgID))

		departments, err := getter.GetDepartmentsByOrgID(r.Context(), orgID)
		if err != nil {
			log.Error("failed to get departments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve departments"))
			return
		}

		log.Info("successfully retrieved departments", slog.Int("count", len(departments)))
		render.JSON(w, r, departments)
	}
}
