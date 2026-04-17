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

type CascadeConfigUpserter interface {
	UpsertCascadeConfig(ctx context.Context, req model.UpsertCascadeConfigRequest) error
}

type CascadeConfigGetter interface {
	GetAllCascadeConfigs(ctx context.Context) ([]model.CascadeConfig, error)
}

type CascadeConfigDeleter interface {
	DeleteCascadeConfig(ctx context.Context, organizationID int64) error
}

func UpsertCascadeConfig(log *slog.Logger, repo CascadeConfigUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.UpsertCascadeConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req model.UpsertCascadeConfigRequest
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

		if err := repo.UpsertCascadeConfig(r.Context(), req); err != nil {
			log.Error("failed to upsert cascade config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save cascade config"))
			return
		}

		log.Info("cascade config upserted", slog.Int64("organization_id", req.OrganizationID))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

func GetCascadeConfigs(log *slog.Logger, repo CascadeConfigGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetCascadeConfigs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		configs, err := repo.GetAllCascadeConfigs(r.Context())
		if err != nil {
			log.Error("failed to get cascade configs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve cascade configs"))
			return
		}

		// Filter by own cascade for non-sc/rais users
		configs = filterCascadeConfigsForCaller(r.Context(), configs)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, configs)
	}
}

// filterCascadeConfigsForCaller returns cascade configs visible to the current
// user. sc/rais see all cascades. Others see only their own cascade (where
// organization_id == claims.OrganizationID).
func filterCascadeConfigsForCaller(ctx context.Context, configs []model.CascadeConfig) []model.CascadeConfig {
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
	filtered := make([]model.CascadeConfig, 0, 1)
	for _, c := range configs {
		if c.OrganizationID == claims.OrganizationID {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func DeleteCascadeConfig(log *slog.Logger, repo CascadeConfigDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.DeleteCascadeConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgID, err := parseIntParam(r, "organization_id")
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required"))
			return
		}

		if err := repo.DeleteCascadeConfig(r.Context(), orgID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("cascade config not found"))
				return
			}
			log.Error("failed to delete cascade config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to delete cascade config"))
			return
		}

		log.Info("cascade config deleted", slog.Int64("organization_id", orgID))

		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
