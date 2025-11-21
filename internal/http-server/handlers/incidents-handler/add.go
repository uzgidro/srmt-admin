package incidents_handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"time"
)

// Request (JSON DTO)
type addRequest struct {
	OrganizationID *int64    `json:"organization_id,omitempty"`
	IncidentTime   time.Time `json:"incident_time" validate:"required"`
	Description    string    `json:"description" validate:"required"`
}

type response struct {
	resp.Response
	ID int64 `json:"id"`
}

type incidentAdder interface {
	AddIncident(ctx context.Context, orgID *int64, incidentTime time.Time, description string, createdByID int64) (int64, error)
}

func Add(log *slog.Logger, adder incidentAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.incident.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := adder.AddIncident(
			r.Context(),
			req.OrganizationID,
			req.IncidentTime,
			req.Description,
			userID,
		)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				orgIDVal := "nil"
				if req.OrganizationID != nil {
					orgIDVal = fmt.Sprintf("%d", *req.OrganizationID)
				}
				log.Warn("organization not found", "org_id", orgIDVal)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add incident", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add incident"))
			return
		}

		log.Info("incident added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, response{Response: resp.OK(), ID: id})
	}
}
