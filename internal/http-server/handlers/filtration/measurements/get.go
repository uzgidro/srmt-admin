package measurements

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type FiltrationMeasurementGetter interface {
	GetFiltrationMeasurements(ctx context.Context, orgID int64, date string) ([]filtration.FiltrationMeasurement, error)
}

type PiezometerMeasurementGetter interface {
	GetPiezometerMeasurements(ctx context.Context, orgID int64, date string) ([]filtration.PiezometerMeasurement, error)
}

type ComparisonDateGetter interface {
	GetComparisonDates(ctx context.Context, orgID int64, date string) (*string, *string, error)
}

type GetResponse struct {
	FiltrationMeasurements []filtration.FiltrationMeasurement `json:"filtration_measurements"`
	PiezometerMeasurements []filtration.PiezometerMeasurement `json:"piezometer_measurements"`
	FilterComparisonDate   *string                            `json:"filter_comparison_date,omitempty"`
	PiezoComparisonDate    *string                            `json:"piezo_comparison_date,omitempty"`
}

func Get(log *slog.Logger, fg FiltrationMeasurementGetter, pg PiezometerMeasurementGetter, cdg ComparisonDateGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.measurements.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgIDStr := r.URL.Query().Get("organization_id")
		if orgIDStr == "" {
			log.Warn("missing required 'organization_id' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'organization_id' parameter"))
			return
		}

		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid organization_id", sl.Err(err), slog.String("organization_id", orgIDStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
			return
		}

		if err := auth.CheckOrgAccess(r.Context(), orgID); err != nil {
			log.Warn("access denied to organization", slog.Int64("org_id", orgID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		date := r.URL.Query().Get("date")
		if date == "" {
			log.Warn("missing required 'date' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}

		filtrationMeasurements, err := fg.GetFiltrationMeasurements(r.Context(), orgID, date)
		if err != nil {
			log.Error("failed to get filtration measurements", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve filtration measurements"))
			return
		}

		piezometerMeasurements, err := pg.GetPiezometerMeasurements(r.Context(), orgID, date)
		if err != nil {
			log.Error("failed to get piezometer measurements", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve piezometer measurements"))
			return
		}

		filterCompDate, piezoCompDate, err := cdg.GetComparisonDates(r.Context(), orgID, date)
		if err != nil {
			log.Error("failed to get comparison dates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve comparison dates"))
			return
		}

		log.Info("successfully retrieved measurements",
			slog.Int64("organization_id", orgID),
			slog.String("date", date),
		)

		render.JSON(w, r, GetResponse{
			FiltrationMeasurements: filtrationMeasurements,
			PiezometerMeasurements: piezometerMeasurements,
			FilterComparisonDate:   filterCompDate,
			PiezoComparisonDate:    piezoCompDate,
		})
	}
}
