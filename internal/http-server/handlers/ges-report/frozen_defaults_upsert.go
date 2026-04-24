package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// FrozenDefaultUpserter is the local dependency contract: persist a frozen
// default and resolve organization parents for cascade-access checks.
type FrozenDefaultUpserter interface {
	UpsertFrozenDefault(ctx context.Context, req model.UpsertFrozenDefaultRequest, userID int64) error
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

// UpsertFrozenDefault freezes a single (organization_id, field_name) pair to a
// constant value that the report layer will use as a sticky carry-forward.
//
// Auth: cascade users are scoped to stations within their own cascade (plan §2.5).
// Aggregate-typed fields (working/repair/modernization) must receive whole numbers
// (plan §7.6); fractional values are rejected with 400 before hitting the DB.
func UpsertFrozenDefault(log *slog.Logger, repo FrozenDefaultUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.UpsertFrozenDefault"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var req model.UpsertFrozenDefaultRequest
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

		// Aggregate fields (working/repair/modernization) are int columns —
		// reject fractional values up-front so the DB never sees nonsense.
		if model.IntegerFreezableFields[req.FieldName] {
			if req.FrozenValue != math.Trunc(req.FrozenValue) {
				log.Warn("non-integer frozen_value for aggregate field",
					slog.String("field_name", req.FieldName),
					slog.Float64("frozen_value", req.FrozenValue),
				)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("frozen_value for aggregate fields must be a whole number"))
				return
			}
		}

		if err := auth.CheckCascadeStationAccess(r.Context(), req.OrganizationID, repo); err != nil {
			log.Warn("cascade access denied for frozen default upsert",
				sl.Err(err),
				slog.Int64("organization_id", req.OrganizationID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("access denied"))
			return
		}

		if err := repo.UpsertFrozenDefault(r.Context(), req, userID); err != nil {
			// Defence-in-depth: the migration's CHECK constraints (allowed
			// field_name list, frozen_value >= 0) duplicate the validator and
			// the integer check above. If somehow bypassed we still surface a
			// 400 instead of an opaque 500.
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				log.Warn("frozen default check constraint violation", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid frozen default value (CHECK constraint)"))
				return
			}
			log.Error("failed to upsert frozen default", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save frozen default"))
			return
		}

		log.Info("frozen default upserted",
			slog.Int64("organization_id", req.OrganizationID),
			slog.String("field_name", req.FieldName),
			slog.Float64("frozen_value", req.FrozenValue),
			slog.Int64("user_id", userID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
