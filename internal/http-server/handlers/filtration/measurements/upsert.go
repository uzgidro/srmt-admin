package measurements

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type AllMeasurementsUpserter interface {
	UpsertAllMeasurements(ctx context.Context, req filtration.UpsertAllMeasurementsRequest) error
}

type UpsertRequest struct {
	OrganizationID int64  `json:"organization_id" validate:"required"`
	Date           string `json:"date" validate:"required"`

	// Current measurements
	FiltrationMeasurements []filtration.FiltrationMeasurementInput `json:"filtration_measurements"`
	PiezometerMeasurements []filtration.PiezometerMeasurementInput `json:"piezometer_measurements"`

	// Historical filtration measurements (optional)
	FilterComparisonDate             *string                                `json:"filter_comparison_date,omitempty"`
	HistoricalFiltrationMeasurements []filtration.FiltrationMeasurementInput `json:"historical_filtration_measurements,omitempty"`

	// Historical piezometer measurements (optional)
	PiezoComparisonDate              *string                                `json:"piezo_comparison_date,omitempty"`
	HistoricalPiezometerMeasurements []filtration.PiezometerMeasurementInput `json:"historical_piezometer_measurements,omitempty"`

	// Explicit clear of comparison_date (removes link to historical data)
	ClearFilterComparisonDate bool `json:"clear_filter_comparison_date,omitempty"`
	ClearPiezoComparisonDate  bool `json:"clear_piezo_comparison_date,omitempty"`
}

func Upsert(log *slog.Logger, upserter AllMeasurementsUpserter) http.HandlerFunc {
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

		parsedDate, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' format (expected YYYY-MM-DD)"))
			return
		}

		// Validate comparison dates and paired fields
		if err := validateComparisonPair(req.FilterComparisonDate, len(req.HistoricalFiltrationMeasurements), parsedDate, "filter"); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(err.Error()))
			return
		}
		if err := validateComparisonPair(req.PiezoComparisonDate, len(req.HistoricalPiezometerMeasurements), parsedDate, "piezo"); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(err.Error()))
			return
		}

		// Clear requires current measurements to be present (comparison_date is on those rows)
		if req.ClearFilterComparisonDate && len(req.FiltrationMeasurements) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("clear_filter_comparison_date requires filtration_measurements"))
			return
		}
		if req.ClearPiezoComparisonDate && len(req.PiezometerMeasurements) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("clear_piezo_comparison_date requires piezometer_measurements"))
			return
		}

		hasCurrent := len(req.FiltrationMeasurements) > 0 || len(req.PiezometerMeasurements) > 0
		hasHistorical := len(req.HistoricalFiltrationMeasurements) > 0 || len(req.HistoricalPiezometerMeasurements) > 0
		if !hasCurrent && !hasHistorical {
			log.Warn("no measurements provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("At least one measurement must be provided"))
			return
		}

		var historicalFilterDate, historicalPiezoDate string
		if req.FilterComparisonDate != nil {
			historicalFilterDate = *req.FilterComparisonDate
		}
		if req.PiezoComparisonDate != nil {
			historicalPiezoDate = *req.PiezoComparisonDate
		}

		if err := upserter.UpsertAllMeasurements(r.Context(), filtration.UpsertAllMeasurementsRequest{
			Date:   req.Date,
			UserID: userID,

			Filtration: req.FiltrationMeasurements,
			Piezometer: req.PiezometerMeasurements,

			FilterComparisonDate: req.FilterComparisonDate,
			PiezoComparisonDate:  req.PiezoComparisonDate,
			ClearFilterCompDate:  req.ClearFilterComparisonDate,
			ClearPiezoCompDate:   req.ClearPiezoComparisonDate,

			HistoricalFilterDate: historicalFilterDate,
			HistoricalFiltration: req.HistoricalFiltrationMeasurements,
			HistoricalPiezoDate:  historicalPiezoDate,
			HistoricalPiezometer: req.HistoricalPiezometerMeasurements,
		}); err != nil {
			log.Error("failed to upsert measurements", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to save measurements"))
			return
		}

		log.Info("measurements upserted successfully",
			slog.String("date", req.Date),
			slog.Int64("user_id", userID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

func validateComparisonPair(compDate *string, itemCount int, currentDate time.Time, label string) error {
	hasDate := compDate != nil && *compDate != ""
	hasItems := itemCount > 0

	if hasDate && !hasItems {
		return errors.New(label + "_comparison_date provided without historical measurements")
	}
	if hasItems && !hasDate {
		return errors.New("historical " + label + " measurements provided without " + label + "_comparison_date")
	}
	if !hasDate {
		return nil
	}

	parsed, err := time.Parse("2006-01-02", *compDate)
	if err != nil {
		return errors.New("invalid '" + label + "_comparison_date' format (expected YYYY-MM-DD)")
	}
	if !parsed.Before(currentDate) {
		return errors.New(label + "_comparison_date must be before date")
	}

	return nil
}
