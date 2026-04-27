package reservoirflood

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-flood"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type HourlyGetter interface {
	GetReservoirFloodHourlyRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]model.HourlyRecord, error)
}

func GetHourly(log *slog.Logger, repo HourlyGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir-flood.GetHourly"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse `date` (YYYY-MM-DD).
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("date query parameter required (YYYY-MM-DD)"))
			return
		}
		day, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}
		// Day window in UTC (24 hours starting from midnight UTC of the given date).
		start := day.UTC()
		end := start.Add(24 * time.Hour)

		// Optional org_id filter from query.
		var orgIDs []int64
		if s := r.URL.Query().Get("organization_id"); s != "" {
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid organization_id"))
				return
			}
			orgIDs = []int64{id}
		}

		records, err := repo.GetReservoirFloodHourlyRange(r.Context(), orgIDs, start, end)
		if err != nil {
			log.Error("failed to get hourly range", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve hourly data"))
			return
		}

		records = filterRecordsForCaller(r.Context(), records)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, records)
	}
}

func filterRecordsForCaller(ctx context.Context, list []model.HourlyRecord) []model.HourlyRecord {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return []model.HourlyRecord{}
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return list
		}
	}
	if claims.OrganizationID == 0 {
		return []model.HourlyRecord{}
	}
	out := make([]model.HourlyRecord, 0, len(list))
	for _, rec := range list {
		if rec.OrganizationID == claims.OrganizationID {
			out = append(out, rec)
		}
	}
	return out
}
