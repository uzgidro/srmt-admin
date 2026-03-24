package comparison

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ComparisonDataGetterV2 interface {
	GetFiltrationOrgIDs(ctx context.Context) ([]int64, error)
	GetOrgFiltrationSummary(ctx context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error)
	GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
}

func GetData(log *slog.Logger, getter ComparisonDataGetterV2) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.comparison.GetData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		date := r.URL.Query().Get("date")
		filterDate := r.URL.Query().Get("filter_date")
		piezoDate := r.URL.Query().Get("piezo_date")

		for _, p := range []struct{ name, val string }{
			{"date", date}, {"filter_date", filterDate}, {"piezo_date", piezoDate},
		} {
			if p.val == "" {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Missing required '"+p.name+"' parameter (format: YYYY-MM-DD)"))
				return
			}
			if _, err := time.Parse("2006-01-02", p.val); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid '"+p.name+"' parameter (format: YYYY-MM-DD)"))
				return
			}
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

		result := make([]filtration.OrgComparisonV2, 0, len(orgIDs))

		for _, orgID := range orgIDs {
			comp, err := buildOrgComparisonV2(r.Context(), getter, orgID, date, filterDate, piezoDate)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					continue
				}
				log.Error("failed to build comparison", sl.Err(err), slog.Int64("org_id", orgID))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to build comparison data"))
				return
			}
			result = append(result, *comp)
		}

		render.JSON(w, r, result)
	}
}

func buildOrgComparisonV2(ctx context.Context, getter ComparisonDataGetterV2, orgID int64, date, filterDate, piezoDate string) (*filtration.OrgComparisonV2, error) {
	// Current snapshot
	summary, err := getter.GetOrgFiltrationSummary(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	level, volume, err := getter.GetReservoirLevelVolume(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	comp := &filtration.OrgComparisonV2{
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

	// Historical filter snapshot
	filterSnap, err := buildSnapshot(ctx, getter, orgID, filterDate)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	comp.HistoricalFilter = filterSnap

	// Historical piezo snapshot — reuse if same date
	if piezoDate == filterDate {
		comp.HistoricalPiezo = filterSnap
	} else {
		piezoSnap, err := buildSnapshot(ctx, getter, orgID, piezoDate)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		comp.HistoricalPiezo = piezoSnap
	}

	return comp, nil
}

func buildSnapshot(ctx context.Context, getter ComparisonDataGetterV2, orgID int64, date string) (*filtration.ComparisonSnapshot, error) {
	summary, err := getter.GetOrgFiltrationSummary(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	level, volume, err := getter.GetReservoirLevelVolume(ctx, orgID, date)
	if err != nil {
		return nil, err
	}

	return &filtration.ComparisonSnapshot{
		Date:        date,
		Level:       level,
		Volume:      volume,
		Locations:   summary.Locations,
		Piezometers: summary.Piezometers,
		PiezoCounts: summary.PiezoCounts,
	}, nil
}
