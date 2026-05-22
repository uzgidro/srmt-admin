package reservoirflood

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-flood"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type HourlyUpserter interface {
	UpsertReservoirFloodHourly(ctx context.Context, items []model.UpsertHourlyRequest, userID int64) error
}

func UpsertHourly(log *slog.Logger, repo HourlyUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir-flood.UpsertHourly"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Warn("no user id in context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var items []model.UpsertHourlyRequest
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

		// Per-item validation + time normalization.
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
			// Parse + truncate to hour.
			t, err := time.Parse(time.RFC3339, items[i].RecordedAt)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      "invalid recorded_at, expected RFC3339",
					"item_index": i,
				})
				return
			}
			normalized := t.UTC().Truncate(time.Hour)
			items[i].RecordedAt = normalized.Format(time.RFC3339)

			// Negative-value defence-in-depth on each Optional[float64].
			// (Validator can't easily reach into the wrapper; do it manually.)
			if violation := negativeMetric(items[i]); violation != "" {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      "metric value must be >= 0: " + violation,
					"item_index": i,
				})
				return
			}
		}

		// Org-bound access check on EVERY item.
		// sc/rais bypass via auth.CheckOrgAccess; reservoir_flood must match own org.
		// Mixed-batch behavior: if ANY item is foreign, the whole batch is rejected
		// (no partial writes). This is enforced by failing the whole request before
		// the repo is called.
		orgIDs := make([]int64, 0, len(items))
		for _, it := range items {
			orgIDs = append(orgIDs, it.OrganizationID)
		}
		if err := auth.CheckOrgAccessBatch(r.Context(), orgIDs); err != nil {
			log.Warn("org access denied for hourly upsert",
				sl.Err(err),
				slog.Int64("user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied to one or more organizations"))
			return
		}

		if err := repo.UpsertReservoirFloodHourly(r.Context(), items, userID); err != nil {
			log.Error("failed to upsert hourly", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save hourly data"))
			return
		}

		log.Info("hourly upserted", slog.Int("count", len(items)), slog.Int64("user_id", userID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

// negativeMetric returns the field name of the first negative metric (if any),
// or "" when all metrics are non-negative or absent. Covers numeric fields with
// non-negative DB CHECK constraints (migrations 000075 + 000077). water_level_m
// uses the Baltic height datum (always positive in operational contexts);
// negative values indicate corrupt input from the frontend. temperature_c is
// intentionally NOT checked here — winter values legitimately go below zero.
func negativeMetric(it model.UpsertHourlyRequest) string {
	if v := it.WaterLevelM.Value; v != nil && *v < 0 {
		return "water_level_m"
	}
	if v := it.WaterVolumeMlnM3.Value; v != nil && *v < 0 {
		return "water_volume_mln_m3"
	}
	if v := it.InflowM3s.Value; v != nil && *v < 0 {
		return "inflow_m3s"
	}
	if v := it.OutflowM3s.Value; v != nil && *v < 0 {
		return "outflow_m3s"
	}
	if v := it.GESFlowM3s.Value; v != nil && *v < 0 {
		return "ges_flow_m3s"
	}
	if v := it.IdleDischargeM3s.Value; v != nil && *v < 0 {
		return "idle_discharge_m3s"
	}
	if v := it.CapacityMwt.Value; v != nil && *v < 0 {
		return "capacity_mwt"
	}
	return ""
}
