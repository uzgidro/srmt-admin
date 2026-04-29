package solar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/solar"
	"srmt-admin/internal/lib/optional"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type DailyDataUpserter interface {
	UpsertSolarDailyData(ctx context.Context, items []model.UpsertDailyDataRequest, userID int64) error
}

func UpsertDailyData(log *slog.Logger, repo DailyDataUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.solar.UpsertDailyData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Warn("no user id in context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var items []model.UpsertDailyDataRequest
		if err := render.DecodeJSON(r.Body, &items); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request format"))
			return
		}
		if len(items) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("data array cannot be empty"))
			return
		}

		// Per-item validation, date parsing, and negative-value check.
		for i := range items {
			if err := validate.Struct(items[i]); err != nil {
				var vErrs validator.ValidationErrors
				errors.As(err, &vErrs)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      "validation failed",
					"item_index": i,
					"details":    vErrs.Error(),
				})
				return
			}
			if _, err := time.Parse("2006-01-02", items[i].Date); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      "invalid date, expected YYYY-MM-DD",
					"item_index": i,
				})
				return
			}
			// Negative-value defence-in-depth on each Optional[float64].
			// (Validator can't easily reach into the wrapper; do it manually.)
			if msg, ok := negativeMetric(items[i]); !ok {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      msg,
					"item_index": i,
				})
				return
			}
		}

		// Org-bound access check on EVERY item.
		// sc/rais bypass via auth.CheckOrgAccessBatch; cascade must match own org.
		// Mixed-batch behavior: if ANY item is foreign, the whole batch is
		// rejected (no partial writes). Broken-account state (cascade caller
		// with OrganizationID == 0) is also handled here — CheckOrgAccessBatch
		// returns ErrNoOrganization, which we translate to 403.
		orgIDs := make([]int64, 0, len(items))
		for _, it := range items {
			orgIDs = append(orgIDs, it.OrganizationID)
		}
		if err := auth.CheckOrgAccessBatch(r.Context(), orgIDs); err != nil {
			log.Warn("org access denied for solar daily-data upsert",
				sl.Err(err),
				slog.Int64("user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied to one or more organizations"))
			return
		}

		if err := repo.UpsertSolarDailyData(r.Context(), items, userID); err != nil {
			log.Error("failed to upsert solar daily-data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save daily data"))
			return
		}

		log.Info("solar daily-data upserted", slog.Int("count", len(items)), slog.Int64("user_id", userID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

// negativeMetric returns ("", true) when all metrics in the item are
// non-negative or absent; otherwise (msg, false) describing the first
// negative field. Mirrors the pattern used in reservoir-flood/hourly_upsert.go
// and ges-report/daily_data.go for Optional[T] wrappers, since validator tags
// can't easily reach into the wrapper.
func negativeMetric(it model.UpsertDailyDataRequest) (string, bool) {
	if msg, ok := checkNonNegativeFloat("generation_kwh", it.OrganizationID, it.GenerationKWh); !ok {
		return msg, false
	}
	if msg, ok := checkNonNegativeFloat("grid_export_kwh", it.OrganizationID, it.GridExportKWh); !ok {
		return msg, false
	}
	return "", true
}

// checkNonNegativeFloat returns ("", true) when the optional field is absent,
// explicit-null, or holds a non-negative value; otherwise (msg, false).
// Float analog of internal/http-server/handlers/ges-report/daily_data.go's
// checkNonNegative for int.
func checkNonNegativeFloat(field string, orgID int64, o optional.Optional[float64]) (string, bool) {
	if !o.Set || o.Value == nil {
		return "", true
	}
	if *o.Value < 0 {
		return fmt.Sprintf("%s must be >= 0 for organization_id=%d, got %v", field, orgID, *o.Value), false
	}
	return "", true
}
