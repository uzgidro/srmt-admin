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

type DeviceTelemetryGetter interface {
	GetDeviceTelemetry(ctx context.Context, stationDBID int64, deviceID string) (*asutp.Envelope, error)
}

func NewGetDevice(log *slog.Logger, getter DeviceTelemetryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.asutp.telemetry.get_device"

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

		deviceID := chi.URLParam(r, "device_id")
		if deviceID == "" {
			log.Warn("missing device_id")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("device_id is required"))
			return
		}

		envelope, err := getter.GetDeviceTelemetry(r.Context(), stationDBID, deviceID)
		if err != nil {
			log.Error("failed to get device telemetry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get telemetry"))
			return
		}

		if envelope == nil {
			log.Info("device telemetry not found", "station_db_id", stationDBID, "device_id", deviceID)
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.NotFound("device telemetry not found"))
			return
		}

		log.Info("device telemetry retrieved", "station_db_id", stationDBID, "device_id", deviceID)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, envelope)
	}
}
