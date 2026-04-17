package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// CascadeDailyWeatherUpserter is the repo interface required by the POST handler.
// UpsertCascadeDailyWeatherBulk writes the partial-update rows in a single
// transaction; GetCascadeConfigByOrgID lets the handler verify each item's
// organization is actually a cascade before writing; GetOrganizationParentID
// is consumed by auth.CheckCascadeStationAccessBatch to enforce per-cascade
// scoping for the "cascade" role.
type CascadeDailyWeatherUpserter interface {
	GetCascadeConfigByOrgID(ctx context.Context, orgID int64) (*model.CascadeConfig, error)
	UpsertCascadeDailyWeatherBulk(ctx context.Context, items []model.UpsertCascadeDailyWeatherRequest) error
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

// CascadeDailyWeatherGetter is the repo interface required by the GET handler.
// GetOrganizationParentID is needed by auth.CheckCascadeStationAccess so the
// "cascade" role is restricted to its own cascade.
type CascadeDailyWeatherGetter interface {
	GetCascadeDailyWeather(ctx context.Context, orgID int64, date string) (*model.CascadeWeather, error)
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

// UpsertCascadeDailyWeather returns the POST handler for manual weather
// corrections on a cascade organization. The body is an array; each item is
// validated, its organization is checked to be a cascade, and then the whole
// batch goes through auth.CheckOrgAccessBatch before the repo write.
func UpsertCascadeDailyWeather(log *slog.Logger, repo CascadeDailyWeatherUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.UpsertCascadeDailyWeather"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if _, err := auth.GetUserID(r.Context()); err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var data []model.UpsertCascadeDailyWeatherRequest
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

		// Per-item: validate, parse date, confirm org is a cascade.
		for i, item := range data {
			if err := validate.Struct(item); err != nil {
				var vErrs validator.ValidationErrors
				errors.As(err, &vErrs)
				log.Error("validation failed", sl.Err(err), slog.Int("item_index", i))
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
			if _, err := repo.GetCascadeConfigByOrgID(r.Context(), item.OrganizationID); err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Warn("organization is not a cascade",
						slog.Int64("organization_id", item.OrganizationID),
						slog.Int("item_index", i),
					)
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, map[string]any{
						"error":           "organization is not a cascade",
						"item_index":      i,
						"organization_id": item.OrganizationID,
					})
					return
				}
				log.Error("failed to verify cascade", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("failed to verify cascade"))
				return
			}
		}

		// Batch cascade access check: sc/rais get full access; cascade users may
		// only edit their own cascade (the cascade self-org case is handled
		// inside CheckCascadeStationAccess); other roles fall back to own-org.
		orgIDs := make([]int64, 0, len(data))
		for _, item := range data {
			orgIDs = append(orgIDs, item.OrganizationID)
		}
		if err := auth.CheckCascadeStationAccessBatch(r.Context(), orgIDs, repo); err != nil {
			log.Warn("cascade access denied for cascade daily weather upsert", sl.Err(err))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied to one or more organizations"))
			return
		}

		if err := repo.UpsertCascadeDailyWeatherBulk(r.Context(), data); err != nil {
			log.Error("failed to upsert cascade daily weather", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save cascade daily weather"))
			return
		}

		log.Info("cascade daily weather upserted", slog.Int("count", len(data)))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

// GetCascadeDailyWeather returns the single-cascade weather row for the given
// organization and date, or a JSON null (HTTP 200) if no row exists — this is
// more convenient for frontend form preloading than a 404.
func GetCascadeDailyWeather(log *slog.Logger, repo CascadeDailyWeatherGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetCascadeDailyWeather"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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
			log.Warn("cascade access denied for cascade daily weather get", sl.Err(err))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied"))
			return
		}

		weather, err := repo.GetCascadeDailyWeather(r.Context(), orgID, date)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusOK)
				render.JSON(w, r, nil)
				return
			}
			log.Error("failed to get cascade daily weather", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get cascade daily weather"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, weather)
	}
}
