package get

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/levelvolume"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Response struct for level volume data
type Response struct {
	resp.Response
	OrganizationID int64   `json:"organization_id"`
	Level          float64 `json:"level"`
	Volume         float64 `json:"volume"`
}

// LevelVolumeGetter interface defining the required repository method
type LevelVolumeGetter interface {
	GetLevelVolume(ctx context.Context, organizationID int64, level float64) (*levelvolume.Model, error)
}

// New creates a handler to get level volume data by organization_id and level
// Query parameters: id (organization_id), level
// Returns organization_id, level, and volume (all 0 if not found)
func New(log *slog.Logger, getter LevelVolumeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.level-volume.get.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Extract query parameters
		organizationIDStr := r.URL.Query().Get("id")
		levelStr := r.URL.Query().Get("level")

		// Validate required parameters
		if organizationIDStr == "" || levelStr == "" {
			log.Warn("missing required parameters")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required parameters: id and level"))
			return
		}

		// Parse organization_id
		organizationID, err := strconv.ParseInt(organizationIDStr, 10, 64)
		if err != nil {
			log.Error("failed to parse organization_id", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
			return
		}

		// Parse level
		level, err := strconv.ParseFloat(levelStr, 64)
		if err != nil {
			log.Error("failed to parse level", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid level"))
			return
		}

		// Get level volume data
		lv, err := getter.GetLevelVolume(r.Context(), organizationID, level)
		if err != nil {
			log.Error("failed to get level volume", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve level volume"))
			return
		}

		log.Info("level volume retrieved",
			slog.Int64("organization_id", lv.OrganizationID),
			slog.Float64("level", lv.Level),
			slog.Float64("volume", lv.Volume),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			Response:       resp.OK(),
			OrganizationID: lv.OrganizationID,
			Level:          lv.Level,
			Volume:         lv.Volume,
		})
	}
}
