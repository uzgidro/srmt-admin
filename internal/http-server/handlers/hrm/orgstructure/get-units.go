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

type UnitTreeGetter interface {
	GetTree(ctx context.Context) ([]orgstructure.OrgUnit, error)
}

func GetUnits(log *slog.Logger, svc UnitTreeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.orgstructure.GetUnits"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		units, err := svc.GetTree(r.Context())
		if err != nil {
			log.Error("failed to get org units", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve org units"))
			return
		}

		render.JSON(w, r, units)
	}
}
