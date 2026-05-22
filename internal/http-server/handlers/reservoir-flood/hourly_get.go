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
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type HourlyGetter interface {
	GetReservoirFloodHourlyRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]model.HourlyRecord, error)
}

func GetHourly(log *slog.Logger, repo HourlyGetter, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir-flood.GetHourly"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse `date` (YYYY-MM-DD) in the configured local timezone, NOT UTC.
		// Pre-fix this used time.Parse which silently treats the input as UTC,
		// so on Asia/Tashkent (UTC+5) records stored at local midnight (UTC
		// 19:00 of the previous day) fell outside "today's" window.
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("date query parameter required (YYYY-MM-DD)"))
			return
		}
		day, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}
		// Default window: full local day [00:00 today, 00:00 next day) → UTC.
		// time.Date(d+1) is DST-safe; +24h would skip/double an hour on DST
		// transitions in zones that have them (Tashkent doesn't, but the rule
		// stays correct for any future loc change).
		start := day.UTC()
		end := time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, loc).UTC()

		// Optional ?hour= narrows the window to one local hour [hh:00, hh+1:00).
		// strconv.Atoi accepts both "0" and "00" — frontend may zero-pad.
		// Validation runs BEFORE the repo so a bad hour never fires a query.
		if s := r.URL.Query().Get("hour"); s != "" {
			h, err := strconv.Atoi(s)
			if err != nil || h < 0 || h > 23 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid hour, expected integer in [0,23]"))
				return
			}
			hourStart := time.Date(day.Year(), day.Month(), day.Day(), h, 0, 0, 0, loc)
			start = hourStart.UTC()
			end = hourStart.Add(time.Hour).UTC()
		}

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

		// Reject broken-account state (non-admin without org) BEFORE the repo,
		// symmetric with the upsert path which uses auth.CheckOrgAccessBatch.
		// Returning 200 with [] would silently mask a misconfigured user.
		if !callerIsAdmin(r.Context()) {
			claims, ok := mwauth.ClaimsFromContext(r.Context())
			if !ok || claims == nil || len(claims.OrganizationIDs) == 0 {
				log.Warn("non-admin caller without organization id")
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("user has no organization assigned"))
				return
			}
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

// filterRecordsForCaller restricts the response to records the caller is
// allowed to see. sc/rais see everything. Other roles (typically
// reservoir_flood) see only records for orgs in their assigned org set. The
// handler MUST have already enforced a non-empty claims.OrganizationIDs
// before calling this for non-admin roles — see GetHourly.
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
	out := make([]model.HourlyRecord, 0, len(list))
	for _, rec := range list {
		if auth.ContainsOrg(claims.OrganizationIDs, rec.OrganizationID) {
			out = append(out, rec)
		}
	}
	return out
}
