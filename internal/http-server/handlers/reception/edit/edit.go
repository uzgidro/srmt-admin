package edit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/reception"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Request - JSON DTO for editing a reception

type editRequest struct {
	Name               *string    `json:"name,omitempty"`
	Together           *string    `json:"together,omitempty"`
	Date               *time.Time `json:"date,omitempty"`
	Description        *string    `json:"description,omitempty"`
	Visitor            *string    `json:"visitor,omitempty"`
	Status             *string    `json:"status,omitempty"` // "default", "true", or "false"
	StatusChangeReason *string    `json:"status_change_reason,omitempty"`
	Informed           *bool      `json:"informed,omitempty"`
}

type editResponse struct {
	resp.Response
}

type receptionEditor interface {
	EditReception(ctx context.Context, receptionID int64, req dto.EditReceptionRequest) error
	GetReceptionByID(ctx context.Context, id int64) (*reception.Model, error)
}

func New(log *slog.Logger, editor receptionEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reception.edit.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get reception ID from URL
		idStr := chi.URLParam(r, "id")
		receptionID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("invalid reception ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid reception ID"))
			return
		}

		// Verify reception exists
		existingReception, err := editor.GetReceptionByID(r.Context(), receptionID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Reception not found"))
				return
			}
			log.Error("failed to get reception", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify reception"))
			return
		}

		// Get user ID from context
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Validate status if provided
		if req.Status != nil {
			status := *req.Status
			if status != "default" && status != "true" && status != "false" {
				log.Error("invalid status value", slog.String("status", status))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status, must be 'default', 'true', or 'false'"))
				return
			}
		}

		// Build storage request
		storageReq := dto.EditReceptionRequest{
			Name:               req.Name,
			Together:           req.Together,
			Date:               req.Date,
			Description:        req.Description,
			Visitor:            req.Visitor,
			Status:             req.Status,
			StatusChangeReason: req.StatusChangeReason,
			UpdatedByID:        userID,
		}

		// Handle 'Informed' logic
		if req.Informed != nil && *req.Informed {
			// Check if status is "default" (either from existing record or update request)
			currentStatus := existingReception.Status
			if req.Status != nil {
				currentStatus = *req.Status
			}

			if currentStatus == "default" {
				log.Warn("cannot set informed=true when status is default")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Reception must be processed (approved/rejected) before being marked as informed"))
				return
			}

			valTrue := true
			storageReq.Informed = &valTrue
			storageReq.InformedByUserID = &userID
		}

		// Update reception
		err = editor.EditReception(r.Context(), receptionID, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Reception not found"))
				return
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reference"))
				return
			}

			log.Error("failed to update reception", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update reception"))
			return
		}

		log.Info("reception updated successfully", slog.Int64("reception_id", receptionID))
		render.JSON(w, r, editResponse{Response: resp.OK()})
	}
}
