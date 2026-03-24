package comparison

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type SimilarDatesGetter interface {
	GetFiltrationOrgIDs(ctx context.Context) ([]int64, error)
	GetReservoirLevelVolume(ctx context.Context, orgID int64, date string) (*float64, *float64, error)
	GetSimilarLevelDates(ctx context.Context, orgID int64, level float64, excludeDate string, limit int) ([]filtration.SimilarDate, error)
	GetOrganizationName(ctx context.Context, orgID int64) (string, error)
}

func GetSimilarDates(log *slog.Logger, getter SimilarDatesGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.comparison.GetSimilarDates"
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

		limit := 20
		if ls := r.URL.Query().Get("limit"); ls != "" {
			if v, err := strconv.Atoi(ls); err == nil && v >= 1 && v <= 100 {
				limit = v
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

		result := make([]filtration.OrgSimilarDates, 0, len(orgIDs))

		for _, orgID := range orgIDs {
			orgName, err := getter.GetOrganizationName(r.Context(), orgID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Warn("org not found, skipping", slog.Int64("org_id", orgID))
					continue
				}
				log.Error("failed to get org name", sl.Err(err), slog.Int64("org_id", orgID))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to retrieve organization"))
				return
			}

			level, volume, err := getter.GetReservoirLevelVolume(r.Context(), orgID, date)
			if err != nil {
				log.Error("failed to get reservoir level/volume", sl.Err(err), slog.Int64("org_id", orgID))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to retrieve reservoir data"))
				return
			}

			entry := filtration.OrgSimilarDates{
				OrganizationID:   orgID,
				OrganizationName: orgName,
				ReferenceDate:    date,
				ReferenceLevel:   level,
				ReferenceVolume:  volume,
				SimilarDates:     make([]filtration.SimilarDate, 0),
			}

			if level != nil {
				dates, err := getter.GetSimilarLevelDates(r.Context(), orgID, *level, date, limit)
				if err != nil {
					log.Error("failed to get similar dates", sl.Err(err), slog.Int64("org_id", orgID))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to retrieve similar dates"))
					return
				}
				entry.SimilarDates = dates
			}

			result = append(result, entry)
		}

		render.JSON(w, r, result)
	}
}
