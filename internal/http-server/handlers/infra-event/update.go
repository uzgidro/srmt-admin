package infraevent

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	CategoryID      *int64  `json:"category_id,omitempty"`
	OrganizationID  *int64  `json:"organization_id,omitempty"`
	OccurredAt      *string `json:"occurred_at,omitempty"`
	RestoredAt      *string `json:"restored_at,omitempty"`
	ClearRestoredAt *bool   `json:"clear_restored_at,omitempty"`
	Description     *string `json:"description,omitempty"`
	Remediation     *string `json:"remediation,omitempty"`
	Notes           *string `json:"notes,omitempty"`
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type eventEditor interface {
	UpdateInfraEvent(ctx context.Context, id int64, req dto.EditInfraEventRequest) error
	UnlinkInfraEventFiles(ctx context.Context, eventID int64) error
	LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error
}

func Update(log *slog.Logger, editor eventEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event.update"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		storageReq := dto.EditInfraEventRequest{
			CategoryID:     req.CategoryID,
			OrganizationID: req.OrganizationID,
			Description:    req.Description,
			Remediation:    req.Remediation,
			Notes:          req.Notes,
		}

		if req.ClearRestoredAt != nil && *req.ClearRestoredAt {
			storageReq.ClearRestoredAt = true
		}

		if req.OccurredAt != nil {
			t, err := time.Parse(time.RFC3339, *req.OccurredAt)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'occurred_at' format, use ISO 8601"))
				return
			}
			storageReq.OccurredAt = &t
		}

		if req.RestoredAt != nil {
			t, err := time.Parse(time.RFC3339, *req.RestoredAt)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'restored_at' format, use ISO 8601"))
				return
			}
			storageReq.RestoredAt = &t
		}

		err = editor.UpdateInfraEvent(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Event not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid category_id or organization_id"))
				return
			}
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("restored_at must be after occurred_at"))
				return
			}
			log.Error("failed to update infra event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update event"))
			return
		}

		// Update files if explicitly provided
		if req.FileIDs != nil {
			if err := editor.UnlinkInfraEventFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink files", sl.Err(err))
			}
			if len(req.FileIDs) > 0 {
				if err := editor.LinkInfraEventFiles(r.Context(), id, req.FileIDs); err != nil {
					log.Error("failed to link files", sl.Err(err))
				}
			}
		}

		log.Info("infra event updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
