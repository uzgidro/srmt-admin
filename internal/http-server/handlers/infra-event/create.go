package infraevent

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addRequest struct {
	CategoryID     int64   `json:"category_id" validate:"required"`
	OrganizationID int64   `json:"organization_id" validate:"required"`
	OccurredAt     string  `json:"occurred_at" validate:"required"`
	RestoredAt     *string `json:"restored_at,omitempty"`
	Description    string  `json:"description" validate:"required"`
	Remediation    *string `json:"remediation,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	FileIDs        []int64 `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type eventAdder interface {
	CreateInfraEvent(ctx context.Context, req dto.AddInfraEventRequest) (int64, error)
	LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error
}

func Create(log *slog.Logger, adder eventAdder) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event.create"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'occurred_at' format, use ISO 8601"))
			return
		}

		var restoredAt *time.Time
		if req.RestoredAt != nil {
			t, err := time.Parse(time.RFC3339, *req.RestoredAt)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'restored_at' format, use ISO 8601"))
				return
			}
			restoredAt = &t
		}

		id, err := adder.CreateInfraEvent(r.Context(), dto.AddInfraEventRequest{
			CategoryID:      req.CategoryID,
			OrganizationID:  req.OrganizationID,
			OccurredAt:      occurredAt,
			RestoredAt:      restoredAt,
			Description:     req.Description,
			Remediation:     req.Remediation,
			Notes:           req.Notes,
			CreatedByUserID: userID,
		})
		if err != nil {
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
			log.Error("failed to create infra event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create event"))
			return
		}

		if len(req.FileIDs) > 0 {
			if err := adder.LinkInfraEventFiles(r.Context(), id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}

		log.Info("infra event created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{Response: resp.OK(), ID: id})
	}
}
