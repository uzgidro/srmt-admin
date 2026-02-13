package orgstructure

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/orgstructure"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type UnitEmployeeGetter interface {
	GetUnitEmployees(ctx context.Context, unitID int64) ([]*orgstructure.OrgEmployee, error)
}

func GetUnitEmployees(log *slog.Logger, svc UnitEmployeeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.orgstructure.GetUnitEmployees"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		employees, err := svc.GetUnitEmployees(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrOrgUnitNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Org unit not found"))
				return
			}
			log.Error("failed to get unit employees", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve unit employees"))
			return
		}

		render.JSON(w, r, employees)
	}
}
