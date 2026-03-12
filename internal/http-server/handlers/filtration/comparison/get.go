package comparison

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ComparisonDataGetter interface {
	GetFiltrationOrgIDs(ctx context.Context) ([]int64, error)
	GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
	GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
	GetClosestLevelDate(ctx context.Context, orgID int64, level float64, excludeDate string) (string, error)
}

func Get(log *slog.Logger, getter ComparisonDataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.comparison.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		date := r.URL.Query().Get("date")
		if date == "" {
			log.Warn("missing required 'date' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			log.Warn("invalid date format", slog.String("date", date))
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

		// Determine org list based on role
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

		result := make([]filtration.OrgComparison, 0, len(orgIDs))

		for _, orgID := range orgIDs {
			comparison, err := buildOrgComparison(r.Context(), getter, orgID, date)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					continue // skip orgs without data
				}
				log.Error("failed to build comparison", sl.Err(err), slog.Int64("org_id", orgID))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to build comparison data"))
				return
			}
			result = append(result, *comparison)
		}

		render.JSON(w, r, result)
	}
}

func buildOrgComparison(ctx context.Context, getter ComparisonDataGetter, orgID int64, date string) (*filtration.OrgComparison, error) {
	// Get current summary (locations, piezometers)
	summary, err := getter.GetOrgFiltrationSummary(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	// Get current level/volume from reservoir_data
	level, volume, err := getter.GetReservoirLevelVolume(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	comparison := &filtration.OrgComparison{
		OrganizationID:   summary.OrganizationID,
		OrganizationName: summary.OrganizationName,
		Current: filtration.ComparisonSnapshot{
			Date:        date,
			Level:       level,
			Volume:      volume,
			Locations:   summary.Locations,
			Piezometers: summary.Piezometers,
			PiezoCounts: summary.PiezoCounts,
		},
	}

	// Find historical date with closest level
	if level != nil {
		histDate, err := getter.GetClosestLevelDate(ctx, orgID, *level, date)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return nil, err
			}
			// No historical data — return without historical snapshot
			return comparison, nil
		}

		// Build historical snapshot
		histSummary, err := getter.GetOrgFiltrationSummary(ctx, orgID, histDate)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return nil, err
			}
			// Org not found for historical date — skip historical snapshot
			return comparison, nil
		}

		histLevel, histVolume, err := getter.GetReservoirLevelVolume(ctx, orgID, histDate)
		if err != nil {
			return nil, err
		}

		comparison.Historical = &filtration.ComparisonSnapshot{
			Date:        histDate,
			Level:       histLevel,
			Volume:      histVolume,
			Locations:   histSummary.Locations,
			Piezometers: histSummary.Piezometers,
			PiezoCounts: histSummary.PiezoCounts,
		}
	}

	return comparison, nil
}
