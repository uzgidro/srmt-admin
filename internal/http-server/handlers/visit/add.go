package visit

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
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addRequest struct {
	OrganizationID  int64  `json:"organization_id" validate:"required"`
	VisitDate       string `json:"visit_date" validate:"required"`
	Description     string `json:"description" validate:"required"`
	ResponsibleName string `json:"responsible_name" validate:"required"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type visitAdder interface {
	AddVisit(ctx context.Context, req dto.AddVisitRequest) (int64, error)
}

func Add(log *slog.Logger, adder visitAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.visit.add.New"
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

		visitDate, err := time.Parse(time.RFC3339, req.VisitDate)
		if err != nil {
			log.Warn("invalid visit_date format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'visit_date' format, use ISO 8601 (e.g., 2024-01-15T10:30:00Z)"))
			return
		}

		id, err := adder.AddVisit(r.Context(), dto.AddVisitRequest{
			OrganizationID:  req.OrganizationID,
			VisitDate:       visitDate,
			Description:     req.Description,
			ResponsibleName: req.ResponsibleName,
			CreatedByUserID: userID,
		})
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization not found", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add visit", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add visit"))
			return
		}

		log.Info("visit added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{Response: resp.OK(), ID: id})
	}
}
