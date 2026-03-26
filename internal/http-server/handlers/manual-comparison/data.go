package manualcomparison

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	manualcomparison "srmt-admin/internal/lib/model/manual-comparison"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ManualComparisonDataGetter interface {
	GetFiltrationOrgIDs(ctx context.Context) ([]int64, error)
	GetManualComparisonBatch(ctx context.Context, orgIDs []int64, date string) (map[int64]*manualcomparison.OrgManualComparison, error)
	GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
	GetPiezometerCounts(ctx context.Context, orgID int64) (*filtration.PiezometerCountsRecord, error)
}

func GetData(log *slog.Logger, getter ManualComparisonDataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.manual-comparison.GetData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		date := r.URL.Query().Get("date")
		if date == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' parameter (format: YYYY-MM-DD)"))
			return
		}

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok || claims == nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		var orgIDs []int64
		isSupervisor := false
		for _, role := range claims.Roles {
			if role == "sc" || role == "rais" {
				isSupervisor = true
				break
			}
		}

		if isSupervisor {
			var err error
			orgIDs, err = getter.GetFiltrationOrgIDs(r.Context())
			if err != nil {
				log.Error("failed to get filtration org IDs", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to retrieve organizations"))
				return
			}
		} else {
			if claims.OrganizationID == 0 {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("No organization assigned"))
				return
			}
			orgIDs = []int64{claims.OrganizationID}
		}

		comparisons, err := buildAllComparisons(r.Context(), getter, orgIDs, date)
		if err != nil {
			log.Error("failed to build comparisons", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to build comparison data"))
			return
		}

		render.JSON(w, r, comparisons)
	}
}

// buildAllComparisons fetches manual comparison data for all orgs using batch queries
// and converts to []filtration.OrgComparisonV2. Shared by GetData and Export handlers.
func buildAllComparisons(ctx context.Context, getter ManualComparisonDataGetter, orgIDs []int64, date string) ([]filtration.OrgComparisonV2, error) {
	mcBatch, err := getter.GetManualComparisonBatch(ctx, orgIDs, date)
	if err != nil {
		return nil, fmt.Errorf("get manual comparison batch: %w", err)
	}

	result := make([]filtration.OrgComparisonV2, 0, len(orgIDs))
	for _, orgID := range orgIDs {
		mc, ok := mcBatch[orgID]
		if !ok {
			continue
		}

		if !hasData(mc) {
			continue
		}

		level, volume, _ := getter.GetReservoirLevelVolume(ctx, orgID, date)

		var piezoCounts filtration.PiezometerCounts
		if countsRec, err := getter.GetPiezometerCounts(ctx, orgID); err == nil {
			piezoCounts = filtration.PiezometerCounts{
				Pressure:    countsRec.PressureCount,
				NonPressure: countsRec.NonPressureCount,
			}
		}

		comp := toComparisonV2(mc, level, volume, piezoCounts)
		result = append(result, comp)
	}
	return result, nil
}

// hasData checks if an org has any actual measurement data entered.
func hasData(mc *manualcomparison.OrgManualComparison) bool {
	for _, f := range mc.Filters {
		if f.FlowRate != nil || f.HistoricalFlowRate != nil {
			return true
		}
	}
	for _, p := range mc.Piezometers {
		if p.Level != nil || p.HistoricalLevel != nil {
			return true
		}
	}
	return false
}

// toComparisonV2 converts manual comparison data into filtration.OrgComparisonV2
// that the Excel generator expects.
func toComparisonV2(
	mc *manualcomparison.OrgManualComparison,
	level, volume *float64,
	piezoCounts filtration.PiezometerCounts,
) filtration.OrgComparisonV2 {
	currentLocs := make([]filtration.LocationReading, len(mc.Filters))
	historicalLocs := make([]filtration.LocationReading, len(mc.Filters))
	for i, f := range mc.Filters {
		loc := filtration.Location{
			ID:        f.LocationID,
			Name:      f.LocationName,
			Norm:      f.Norm,
			SortOrder: f.SortOrder,
		}
		currentLocs[i] = filtration.LocationReading{
			Location: loc,
			FlowRate: f.FlowRate,
		}
		historicalLocs[i] = filtration.LocationReading{
			Location: loc,
			FlowRate: f.HistoricalFlowRate,
		}
	}

	currentPiezos := make([]filtration.PiezoReading, len(mc.Piezometers))
	historicalPiezos := make([]filtration.PiezoReading, len(mc.Piezometers))
	for i, p := range mc.Piezometers {
		piezo := filtration.Piezometer{
			ID:        p.PiezometerID,
			Name:      p.PiezometerName,
			Norm:      p.Norm,
			SortOrder: p.SortOrder,
		}
		currentPiezos[i] = filtration.PiezoReading{
			Piezometer: piezo,
			Level:      p.Level,
			Anomaly:    p.Anomaly,
		}
		historicalPiezos[i] = filtration.PiezoReading{
			Piezometer: piezo,
			Level:      p.HistoricalLevel,
		}
	}

	comp := filtration.OrgComparisonV2{
		OrganizationID:   mc.OrganizationID,
		OrganizationName: mc.OrganizationName,
		Current: filtration.ComparisonSnapshot{
			Date:        mc.Date,
			Level:       level,
			Volume:      volume,
			Locations:   currentLocs,
			Piezometers: currentPiezos,
			PiezoCounts: piezoCounts,
		},
	}

	// Historical filter snapshot
	if mc.HistoricalFilterDate != "" {
		comp.HistoricalFilter = &filtration.ComparisonSnapshot{
			Date:        mc.HistoricalFilterDate,
			Locations:   historicalLocs,
			Piezometers: historicalPiezos,
			PiezoCounts: piezoCounts,
		}
	}

	// Historical piezo snapshot — always separate instance to avoid pointer aliasing
	if mc.HistoricalPiezoDate != "" {
		comp.HistoricalPiezo = &filtration.ComparisonSnapshot{
			Date:        mc.HistoricalPiezoDate,
			Locations:   historicalLocs,
			Piezometers: historicalPiezos,
			PiezoCounts: piezoCounts,
		}
	}

	return comp
}
