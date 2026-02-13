package access

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/access"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ZoneAllGetter interface {
	GetAllZones(ctx context.Context) ([]*access.AccessZone, error)
}

func GetZones(log *slog.Logger, svc ZoneAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetZones"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		zones, err := svc.GetAllZones(r.Context())
		if err != nil {
			log.Error("failed to get access zones", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access zones"))
			return
		}

		render.JSON(w, r, zones)
	}
}
