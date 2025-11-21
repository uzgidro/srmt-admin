package reservoirdevicesummary

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type reservoirDeviceSummaryPatcher interface {
	PatchReservoirDeviceSummary(ctx context.Context, req dto.PatchReservoirDeviceSummaryRequest, updatedByUserID int64) error
}

func Patch(log *slog.Logger, patcher reservoirDeviceSummaryPatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirdevicesummary.Patch"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("User not authenticated"))
			return
		}

		var req dto.PatchReservoirDeviceSummaryRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if len(req.Updates) == 0 {
			log.Warn("no updates provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("No updates provided"))
			return
		}

		// Validate that each update has required fields
		for i, update := range req.Updates {
			if update.OrganizationID == 0 {
				log.Warn("missing organization_id in update", slog.Int("index", i))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Missing organization_id in one or more updates"))
				return
			}
			if update.DeviceTypeName == "" {
				log.Warn("missing device_type_name in update", slog.Int("index", i))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Missing device_type_name in one or more updates"))
				return
			}
		}

		err = patcher.PatchReservoirDeviceSummary(r.Context(), req, userID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("reservoir device summary not found")
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("One or more reservoir device summaries not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
				return
			}
			log.Error("failed to update reservoir device summaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update reservoir device summaries"))
			return
		}

		log.Info("reservoir device summaries updated", slog.Int("count", len(req.Updates)), slog.Int64("user_id", userID))
		render.JSON(w, r, resp.OK())
	}
}
