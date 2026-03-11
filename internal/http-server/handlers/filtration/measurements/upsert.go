package measurements

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type FiltrationMeasurementUpserter interface {
	UpsertFiltrationMeasurements(ctx context.Context, date string, items []filtration.FiltrationMeasurementInput, userID int64) error
}

type PiezometerMeasurementUpserter interface {
	UpsertPiezometerMeasurements(ctx context.Context, date string, items []filtration.PiezometerMeasurementInput, userID int64) error
}

type UpsertRequest struct {
	OrganizationID         int64                                   `json:"organization_id" validate:"required"`
	Date                   string                                  `json:"date" validate:"required"`
	FiltrationMeasurements []filtration.FiltrationMeasurementInput  `json:"filtration_measurements"`
	PiezometerMeasurements []filtration.PiezometerMeasurementInput  `json:"piezometer_measurements"`
}

func Upsert(log *slog.Logger, fu FiltrationMeasurementUpserter, pu PiezometerMeasurementUpserter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.measurements.Upsert"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req UpsertRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		validate := validator.New()
		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
			log.Warn("access denied to organization", slog.Int64("org_id", req.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		if len(req.FiltrationMeasurements) == 0 && len(req.PiezometerMeasurements) == 0 {
			log.Warn("no measurements provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("At least one measurement must be provided"))
			return
		}

		if len(req.FiltrationMeasurements) > 0 {
			if err := fu.UpsertFiltrationMeasurements(r.Context(), req.Date, req.FiltrationMeasurements, userID); err != nil {
				log.Error("failed to upsert filtration measurements", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to save filtration measurements"))
				return
			}
		}

		if len(req.PiezometerMeasurements) > 0 {
			if err := pu.UpsertPiezometerMeasurements(r.Context(), req.Date, req.PiezometerMeasurements, userID); err != nil {
				log.Error("failed to upsert piezometer measurements", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to save piezometer measurements"))
				return
			}
		}

		log.Info("measurements upserted successfully",
			slog.String("date", req.Date),
			slog.Int64("user_id", userID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
