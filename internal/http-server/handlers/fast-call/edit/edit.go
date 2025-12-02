package edit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/fast_call"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	ContactID *int64 `json:"contact_id,omitempty"`
	Position  *int   `json:"position,omitempty"`
}

type editResponse struct {
	resp.Response
}

type fastCallEditor interface {
	EditFastCall(ctx context.Context, fastCallID int64, req dto.EditFastCallRequest) error
	GetFastCallByID(ctx context.Context, id int64) (*fast_call.Model, error)
}

func New(log *slog.Logger, editor fastCallEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.fast_call.edit.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get fast call ID from URL
		idStr := chi.URLParam(r, "id")
		fastCallID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("invalid fast call ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid fast call ID"))
			return
		}

		// Verify fast call exists
		_, err = editor.GetFastCallByID(r.Context(), fastCallID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Fast call not found"))
				return
			}
			log.Error("failed to get fast call", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify fast call"))
			return
		}

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Build storage request
		storageReq := dto.EditFastCallRequest{
			ContactID: req.ContactID,
			Position:  req.Position,
		}

		// Update fast call
		err = editor.EditFastCall(r.Context(), fastCallID, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Fast call not found"))
				return
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Contact with this ID does not exist"))
				return
			}

			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate position", slog.Any("position", req.Position))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Fast call with this position already exists"))
				return
			}

			log.Error("failed to update fast call", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update fast call"))
			return
		}

		log.Info("fast call updated successfully", slog.Int64("fast_call_id", fastCallID))
		render.JSON(w, r, editResponse{Response: resp.OK()})
	}
}
