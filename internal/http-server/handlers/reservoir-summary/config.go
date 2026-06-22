package reservoirsummary

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// Local interfaces — handlers depend on what they call, nothing more.

type ConfigUpserter interface {
	UpsertReservoirSummaryConfig(ctx context.Context, req model.UpsertReservoirSummaryConfigRequest) error
}

type ConfigGetter interface {
	GetAllReservoirSummaryConfigs(ctx context.Context) ([]model.ReservoirSummaryConfig, error)
}

type ConfigDeleter interface {
	DeleteReservoirSummaryConfig(ctx context.Context, organizationID int64) error
}

// UpsertConfig handles POST /reservoir-summary/config — create or update
// (by organization_id) a single config row. sc/rais only.
func UpsertConfig(log *slog.Logger, repo ConfigUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummary.UpsertConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req model.UpsertReservoirSummaryConfigRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request format"))
			return
		}

		// Default to "static" before validation so legacy clients (no
		// volume_source field) keep working, and so the repo always writes
		// a CHECK-compatible value into reservoir_summary_config.
		if req.VolumeSource == "" {
			req.VolumeSource = "static"
		}

		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		if err := repo.UpsertReservoirSummaryConfig(r.Context(), req); err != nil {
			// organization_id has a FK to organizations(id); a bad ID from
			// the UI surfaces as 422 with a real message instead of 500.
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				render.Status(r, http.StatusUnprocessableEntity)
				render.JSON(w, r, resp.BadRequest("organization does not exist"))
				return
			}
			log.Error("failed to upsert reservoir-summary config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save config"))
			return
		}

		log.Info("reservoir-summary config upserted", slog.Int64("organization_id", req.OrganizationID))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

// GetConfigs handles GET /reservoir-summary/config. sc/rais see all rows;
// other roles see only rows whose organization_id is in their claims.
func GetConfigs(log *slog.Logger, repo ConfigGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummary.GetConfigs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		configs, err := repo.GetAllReservoirSummaryConfigs(r.Context())
		if err != nil {
			log.Error("failed to get reservoir-summary configs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve configs"))
			return
		}

		configs = filterConfigsForCaller(r.Context(), configs)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, configs)
	}
}

// filterConfigsForCaller is the same pattern as ges-report's
// filterCascadeConfigsForCaller: sc/rais see everything, other roles see
// only rows belonging to their own organizations.
func filterConfigsForCaller(ctx context.Context, configs []model.ReservoirSummaryConfig) []model.ReservoirSummaryConfig {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return []model.ReservoirSummaryConfig{}
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return configs
		}
	}
	if len(claims.OrganizationIDs) == 0 {
		return []model.ReservoirSummaryConfig{}
	}
	filtered := make([]model.ReservoirSummaryConfig, 0, 1)
	for _, c := range configs {
		if auth.ContainsOrg(claims.OrganizationIDs, c.OrganizationID) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// DeleteConfig handles DELETE /reservoir-summary/config?organization_id=...
// sc/rais only. 404 if no row matched.
func DeleteConfig(log *slog.Logger, repo ConfigDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummary.DeleteConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgIDStr := r.URL.Query().Get("organization_id")
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil || orgID <= 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required and must be a positive integer"))
			return
		}

		if err := repo.DeleteReservoirSummaryConfig(r.Context(), orgID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("config not found"))
				return
			}
			log.Error("failed to delete reservoir-summary config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to delete config"))
			return
		}

		log.Info("reservoir-summary config deleted", slog.Int64("organization_id", orgID))

		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
