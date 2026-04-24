package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// FrozenDefaultDeleter is the local dependency contract: remove a frozen
// default and resolve organization parents for cascade-access checks.
type FrozenDefaultDeleter interface {
	DeleteFrozenDefault(ctx context.Context, orgID int64, field string) error
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

// DeleteFrozenDefault removes a single (organization_id, field_name) freeze
// entry. The endpoint is idempotent: a missing row is treated as success
// (no 404) so callers don't have to special-case it.
//
// Auth mirrors UpsertFrozenDefault: cascade users may only delete entries
// for stations within their cascade.
func DeleteFrozenDefault(log *slog.Logger, repo FrozenDefaultDeleter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.DeleteFrozenDefault"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req model.DeleteFrozenDefaultRequest
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

		if err := auth.CheckCascadeStationAccess(r.Context(), req.OrganizationID, repo); err != nil {
			log.Warn("cascade access denied for frozen default delete",
				sl.Err(err),
				slog.Int64("organization_id", req.OrganizationID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied"))
			return
		}

		if err := repo.DeleteFrozenDefault(r.Context(), req.OrganizationID, req.FieldName); err != nil {
			log.Error("failed to delete frozen default", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to delete frozen default"))
			return
		}

		log.Info("frozen default deleted",
			slog.Int64("organization_id", req.OrganizationID),
			slog.String("field_name", req.FieldName),
		)

		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
