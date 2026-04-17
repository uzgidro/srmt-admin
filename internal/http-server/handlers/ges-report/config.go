package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type ConfigUpserter interface {
	UpsertGESConfig(ctx context.Context, req model.UpsertConfigRequest) error
}

type ConfigGetter interface {
	GetAllGESConfigs(ctx context.Context) ([]model.Config, error)
}

type ConfigDeleter interface {
	DeleteGESConfig(ctx context.Context, organizationID int64) error
}

func UpsertConfig(log *slog.Logger, repo ConfigUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.UpsertConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req model.UpsertConfigRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request format"))
			return
		}

		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		if err := repo.UpsertGESConfig(r.Context(), req); err != nil {
			log.Error("failed to upsert ges config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save config"))
			return
		}

		log.Info("ges config upserted", slog.Int64("organization_id", req.OrganizationID))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

func GetConfigs(log *slog.Logger, repo ConfigGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetConfigs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		configs, err := repo.GetAllGESConfigs(r.Context())
		if err != nil {
			log.Error("failed to get ges configs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve configs"))
			return
		}

		// Filter by cascade for non-sc/rais users
		configs = filterGESConfigsForCaller(r.Context(), configs)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, configs)
	}
}

// filterGESConfigsForCaller returns configs visible to the current user.
// sc/rais see all. Others see only stations whose cascade_id matches their
// organization_id (i.e. stations in their own cascade, plus their own org).
func filterGESConfigsForCaller(ctx context.Context, configs []model.Config) []model.Config {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return configs
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return configs
		}
	}
	if claims.OrganizationID == 0 {
		return configs
	}
	filtered := make([]model.Config, 0, len(configs))
	for _, c := range configs {
		if c.OrganizationID == claims.OrganizationID {
			filtered = append(filtered, c)
			continue
		}
		if c.CascadeID != nil && *c.CascadeID == claims.OrganizationID {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func DeleteConfig(log *slog.Logger, repo ConfigDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.DeleteConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgID, err := parseIntParam(r, "organization_id")
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required"))
			return
		}

		if err := repo.DeleteGESConfig(r.Context(), orgID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("config not found"))
				return
			}
			log.Error("failed to delete ges config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to delete config"))
			return
		}

		log.Info("ges config deleted", slog.Int64("organization_id", orgID))

		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
