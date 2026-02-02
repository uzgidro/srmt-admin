package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/asutp"
)

type StationTelemetryGetter interface {
	GetStationTelemetry(ctx context.Context, stationDBID int64) ([]*asutp.Envelope, error)
}

func NewGetStation(log *slog.Logger, getter StationTelemetryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.asutp.telemetry.get_station"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		stationDBID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid station id", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid station id"))
			return
		}

		envelopes, err := getter.GetStationTelemetry(r.Context(), stationDBID)
		if err != nil {
			log.Error("failed to get station telemetry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get telemetry"))
			return
		}

		log.Info("station telemetry retrieved", "station_db_id", stationDBID, "devices_count", len(envelopes))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, envelopes)
	}
}
