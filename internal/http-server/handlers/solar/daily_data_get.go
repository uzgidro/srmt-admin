package solar

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/solar"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DailyDataGetter interface {
	GetSolarDailyDataRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]model.DailyData, error)
}

func GetDailyData(log *slog.Logger, repo DailyDataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.solar.GetDailyData"
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
		// Day window in UTC: half-open [start, end) — start at midnight UTC,
		// end at start-of-next-day UTC.
		start := day.UTC()
		end := start.Add(24 * time.Hour)

		// Validate optional `organization_id` query param early so callers get a
		// clean 400 on bad input. The value itself is only a HINT for sc/rais —
		// it is NOT trusted for cascade callers (see below).
		if s := r.URL.Query().Get("organization_id"); s != "" {
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid organization_id"))
				return
			}
		}

		// Build orgIDs slice based on caller's role.
		// sc/rais: nil → repo returns all orgs. They may also pass a hint via
		//          query param, which we honor for their convenience.
		// other roles (cascade/etc.): caller's own org ONLY. The query param
		//          is IGNORED here as a source of truth — it remains a hint
		//          that is overridden, and the post-repo defence-in-depth
		//          filter strips any foreign-org records that slipped through.
		var orgIDs []int64
		if callerIsAdmin(r.Context()) {
			if s := r.URL.Query().Get("organization_id"); s != "" {
				id, _ := strconv.ParseInt(s, 10, 64)
				orgIDs = []int64{id}
			}
		} else {
			// Reject broken-account state (non-admin without org) BEFORE the
			// repo: returning 200 with [] would silently mask a misconfigured
			// user. Symmetric with the upsert path which uses CheckOrgAccessBatch.
			claims, ok := mwauth.ClaimsFromContext(r.Context())
			if !ok || claims == nil || len(claims.OrganizationIDs) == 0 {
				log.Warn("non-admin caller without organization id")
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("user has no organization assigned"))
				return
			}
			orgIDs = claims.OrganizationIDs
		}

		records, err := repo.GetSolarDailyDataRange(r.Context(), orgIDs, start, end)
		if err != nil {
			log.Error("failed to get solar daily-data range", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve daily data"))
			return
		}

		// Defence-in-depth: even though we passed the cascade caller's own org
		// to the repo, strip any record whose org doesn't match. Protects
		// against future repo bugs and against a misused `organization_id`
		// hint the cascade caller might have supplied.
		records = filterDailyDataForCaller(r.Context(), records)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, records)
	}
}
