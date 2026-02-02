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

type TelemetrySaver interface {
	SaveTelemetry(ctx context.Context, stationDBID int64, env *asutp.Envelope) error
}

type PostResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

func NewPost(log *slog.Logger, saver TelemetrySaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.asutp.telemetry.post"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		stationDBID, err := strconv.ParseInt(chi.URLParam(r, "station_db_id"), 10, 64)
		if err != nil {
			log.Warn("invalid station_db_id", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid station_db_id"))
			return
		}

		var env asutp.Envelope
		if err := render.DecodeJSON(r.Body, &env); err != nil {
			log.Error("failed to parse request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request body"))
			return
		}

		if env.ID == "" || env.DeviceID == "" {
			log.Warn("missing required fields", "id", env.ID, "device_id", env.DeviceID)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("id and device_id are required"))
			return
		}

		if err := saver.SaveTelemetry(r.Context(), stationDBID, &env); err != nil {
			log.Error("failed to save telemetry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save telemetry"))
			return
		}

		log.Info("telemetry saved", "station_db_id", stationDBID, "device_id", env.DeviceID)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, PostResponse{
			Status: "ok",
			ID:     env.ID,
		})
	}
}
