package gesreport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/optional"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type DailyDataUpserter interface {
	UpsertGESDailyData(ctx context.Context, items []model.UpsertDailyDataRequest, userID int64) error
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
	GetGESConfigsTotalAggregates(ctx context.Context, orgIDs []int64) (map[int64]int, error)
	GetGESDailyAggregatesBatch(ctx context.Context, orgIDs []int64, date string) (map[int64]model.AggregateCounts, error)
	GetGESConfigsMaxDailyProduction(ctx context.Context) (map[int64]float64, error)
	GetGESDailyProductionsBatch(ctx context.Context, orgIDs []int64, date string) (map[int64]float64, error)
}

type DailyDataGetter interface {
	GetGESDailyData(ctx context.Context, organizationID int64, date string) (*model.DailyData, error)
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

func UpsertDailyData(log *slog.Logger, repo DailyDataUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.UpsertDailyData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var data []model.UpsertDailyDataRequest
		if err := render.DecodeJSON(r.Body, &data); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request format"))
			return
		}

		if len(data) == 0 {
			log.Warn("empty data array received")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("data array cannot be empty"))
			return
		}

		// Per-item validation
		for i, item := range data {
			if err := validate.Struct(item); err != nil {
				var vErrs validator.ValidationErrors
				errors.As(err, &vErrs)
				log.Error("validation failed",
					sl.Err(err),
					slog.Int("item_index", i),
				)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      "validation failed",
					"item_index": i,
					"details":    vErrs.Error(),
				})
				return
			}
			if _, err := time.Parse("2006-01-02", item.Date); err != nil {
				log.Error("invalid date format",
					sl.Err(err),
					slog.Int("item_index", i),
					slog.String("date", item.Date),
				)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      "invalid date format, expected YYYY-MM-DD",
					"item_index": i,
				})
				return
			}
		}

		// Batch organization access check
		orgIDs := make([]int64, 0, len(data))
		for _, item := range data {
			orgIDs = append(orgIDs, item.OrganizationID)
		}
		if err := auth.CheckCascadeStationAccessBatch(r.Context(), orgIDs, repo); err != nil {
			log.Warn("cascade access denied for ges daily data upsert", sl.Err(err))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied to one or more organizations"))
			return
		}

		// Aggregate validation: per-field non-negative + sum ≤ ges_config.total.
		// Run after auth so foreign-org probes can't inspect cap values via 400s.
		if status, msg, err := validateAggregates(r.Context(), data, repo); status != 0 {
			if err != nil {
				log.Error("aggregate validation lookup failed", sl.Err(err))
			} else {
				log.Warn("aggregate validation rejected request", slog.String("reason", msg))
			}
			render.Status(r, status)
			if status == http.StatusInternalServerError {
				render.JSON(w, r, resp.InternalServerError(msg))
			} else {
				render.JSON(w, r, resp.BadRequest(msg))
			}
			return
		}

		// Production cap validation: effective daily_production_mln_kwh ≤
		// ges_config.max_daily_production_mln_kwh. Stations without a positive
		// cap (absent from the map) are unconstrained. Runs after auth for the
		// same probe-resistance reason as validateAggregates.
		if status, msg, err := validateProductionCap(r.Context(), data, repo); status != 0 {
			if err != nil {
				log.Error("production cap validation lookup failed", sl.Err(err))
			} else {
				log.Warn("production cap validation rejected request", slog.String("reason", msg))
			}
			render.Status(r, status)
			if status == http.StatusInternalServerError {
				render.JSON(w, r, resp.InternalServerError(msg))
			} else {
				render.JSON(w, r, resp.BadRequest(msg))
			}
			return
		}

		if err := repo.UpsertGESDailyData(r.Context(), data, userID); err != nil {
			log.Error("failed to upsert ges daily data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save daily data"))
			return
		}

		log.Info("ges daily data upserted",
			slog.Int("count", len(data)),
			slog.Int64("user_id", userID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

func GetDailyData(log *slog.Logger, repo DailyDataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetDailyData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgID, err := parseIntParam(r, "organization_id")
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required"))
			return
		}

		date := r.URL.Query().Get("date")
		if date == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("date is required (YYYY-MM-DD)"))
			return
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}

		if err := auth.CheckCascadeStationAccess(r.Context(), orgID, repo); err != nil {
			log.Warn("cascade access denied for ges daily data get", sl.Err(err))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied"))
			return
		}

		data, err := repo.GetGESDailyData(r.Context(), orgID, date)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusOK)
				render.JSON(w, r, nil)
				return
			}
			log.Error("failed to get ges daily data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve daily data"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, data)
	}
}

// validateAggregates enforces two rules on UpsertDailyData payloads:
//   1. each provided working/repair/modernization value is non-negative
//      (Set==true && Value!=nil && *Value < 0 → 400);
//   2. for every (organization_id, date) tuple, the effective aggregate sum
//      after applying the request onto the existing DB row does not exceed
//      ges_config.total_aggregates (when configured).
//
// Returns a (status, message, err) triple where status==0 means OK. status
// http.StatusBadRequest carries the user-facing message; status
// http.StatusInternalServerError carries a generic message and a non-nil err
// for logging.
func validateAggregates(
	ctx context.Context,
	data []model.UpsertDailyDataRequest,
	repo DailyDataUpserter,
) (int, string, error) {
	// 1. Per-field non-negative check on what was actually provided.
	for _, item := range data {
		if msg, ok := checkNonNegative("working_aggregates", item.OrganizationID, item.WorkingAggregates); !ok {
			return http.StatusBadRequest, msg, nil
		}
		if msg, ok := checkNonNegative("repair_aggregates", item.OrganizationID, item.RepairAggregates); !ok {
			return http.StatusBadRequest, msg, nil
		}
		if msg, ok := checkNonNegative("modernization_aggregates", item.OrganizationID, item.ModernizationAggregates); !ok {
			return http.StatusBadRequest, msg, nil
		}
	}

	// 2. Sum check requires totals from ges_config and current DB values for
	// any field the request omitted. Skip work entirely when no item touches
	// any aggregate field — nothing can change, so the sum cannot grow.
	uniqueOrgIDs := uniqueOrgs(data)
	totals, err := repo.GetGESConfigsTotalAggregates(ctx, uniqueOrgIDs)
	if err != nil {
		return http.StatusInternalServerError, "failed to load aggregate caps", err
	}

	// Group orgIDs by date so we issue one current-values query per distinct
	// date in the request (typically just one).
	byDate := groupOrgsByDate(data)
	currents := make(map[string]map[int64]model.AggregateCounts, len(byDate))
	for date, orgs := range byDate {
		cur, err := repo.GetGESDailyAggregatesBatch(ctx, orgs, date)
		if err != nil {
			return http.StatusInternalServerError, "failed to load current aggregates", err
		}
		currents[date] = cur
	}

	for _, item := range data {
		total, hasCap := totals[item.OrganizationID]
		if !hasCap {
			// No ges_config row → trigger also skips, so we skip too.
			continue
		}
		cur := currents[item.Date][item.OrganizationID] // zero-value when missing
		w := effective(item.WorkingAggregates, cur.Working)
		rep := effective(item.RepairAggregates, cur.Repair)
		mod := effective(item.ModernizationAggregates, cur.Modernization)
		sum := w + rep + mod
		if sum > total {
			return http.StatusBadRequest, fmt.Sprintf(
				"aggregates sum exceeds total for organization_id=%d: %d+%d+%d=%d > %d",
				item.OrganizationID, w, rep, mod, sum, total,
			), nil
		}
	}
	return 0, "", nil
}

// validateProductionCap enforces ges_config.max_daily_production_mln_kwh on
// the effective daily_production_mln_kwh that will land in the row after the
// upsert. Effective semantics mirror validateAggregates:
//   - field absent (Set=false)            → preserve current DB value
//   - field present with non-nil number   → use that number
//   - field present but null              → COALESCE writes 0
//
// A station without a positive cap (absent from the map per repo contract,
// which already filters max==0) is unrestricted, preserving backwards
// compatibility with previously-unconfigured stations.
func validateProductionCap(
	ctx context.Context,
	data []model.UpsertDailyDataRequest,
	repo DailyDataUpserter,
) (int, string, error) {
	maxMap, err := repo.GetGESConfigsMaxDailyProduction(ctx)
	if err != nil {
		return http.StatusInternalServerError, "failed to load production caps", err
	}
	if len(maxMap) == 0 {
		return 0, "", nil // no station has a cap → nothing to check
	}

	// Group by date the orgs that omit the production field AND have a cap —
	// only those rows need a preserve-DB lookup.
	preserveByDate := make(map[string][]int64)
	preserveSeen := make(map[string]map[int64]struct{})
	for _, item := range data {
		if _, capped := maxMap[item.OrganizationID]; !capped {
			continue
		}
		if item.DailyProductionMlnKWh.Set {
			continue
		}
		s, ok := preserveSeen[item.Date]
		if !ok {
			s = make(map[int64]struct{})
			preserveSeen[item.Date] = s
		}
		if _, dup := s[item.OrganizationID]; dup {
			continue
		}
		s[item.OrganizationID] = struct{}{}
		preserveByDate[item.Date] = append(preserveByDate[item.Date], item.OrganizationID)
	}

	// Issue one batch query per distinct date.
	currentProd := make(map[string]map[int64]float64, len(preserveByDate))
	for date, orgs := range preserveByDate {
		cur, err := repo.GetGESDailyProductionsBatch(ctx, orgs, date)
		if err != nil {
			return http.StatusInternalServerError, "failed to load current production", err
		}
		currentProd[date] = cur
	}

	for _, item := range data {
		cap, capped := maxMap[item.OrganizationID]
		if !capped {
			continue
		}
		var effectiveVal float64
		switch {
		case !item.DailyProductionMlnKWh.Set:
			effectiveVal = currentProd[item.Date][item.OrganizationID] // 0 if no row
		case item.DailyProductionMlnKWh.Value == nil:
			effectiveVal = 0
		default:
			effectiveVal = *item.DailyProductionMlnKWh.Value
		}
		if effectiveVal > cap {
			return http.StatusBadRequest, fmt.Sprintf(
				"daily_production_mln_kwh exceeds max for organization_id=%d: %g > %g",
				item.OrganizationID, effectiveVal, cap,
			), nil
		}
	}
	return 0, "", nil
}

// effective returns the value the upsert will actually write for an aggregate
// column, mirroring the SQL semantics of UpsertGESDailyData:
//   - field absent (Set=false)            → preserve current DB value
//   - field present with non-nil number   → use that number
//   - field present but null (Set=true,
//     Value=nil) → COALESCE($N,0) writes 0
func effective(o optional.Optional[int], current int) int {
	if !o.Set {
		return current
	}
	if o.Value == nil {
		return 0
	}
	return *o.Value
}

// checkNonNegative returns (msg, false) when the field carries an explicit
// negative number; otherwise (empty, true). Absent fields and explicit nulls
// pass — they cannot represent a negative value.
func checkNonNegative(field string, orgID int64, o optional.Optional[int]) (string, bool) {
	if !o.Set || o.Value == nil {
		return "", true
	}
	if *o.Value < 0 {
		return fmt.Sprintf("%s must be >= 0 for organization_id=%d, got %d", field, orgID, *o.Value), false
	}
	return "", true
}

// uniqueOrgs returns the distinct organization IDs across the request,
// preserving insertion order for deterministic logs and queries.
func uniqueOrgs(data []model.UpsertDailyDataRequest) []int64 {
	seen := make(map[int64]struct{}, len(data))
	out := make([]int64, 0, len(data))
	for _, item := range data {
		if _, ok := seen[item.OrganizationID]; ok {
			continue
		}
		seen[item.OrganizationID] = struct{}{}
		out = append(out, item.OrganizationID)
	}
	return out
}

// groupOrgsByDate clusters distinct organization IDs by their request date so
// we can issue one batch query per date for current aggregate values.
func groupOrgsByDate(data []model.UpsertDailyDataRequest) map[string][]int64 {
	out := make(map[string][]int64)
	seen := make(map[string]map[int64]struct{})
	for _, item := range data {
		s, ok := seen[item.Date]
		if !ok {
			s = make(map[int64]struct{})
			seen[item.Date] = s
		}
		if _, dup := s[item.OrganizationID]; dup {
			continue
		}
		s[item.OrganizationID] = struct{}{}
		out[item.Date] = append(out[item.Date], item.OrganizationID)
	}
	return out
}
