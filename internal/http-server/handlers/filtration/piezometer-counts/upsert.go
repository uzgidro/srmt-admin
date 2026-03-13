package piezometercounts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type PiezometerCountsUpserter interface {
	UpsertPiezometerCounts(ctx context.Context, req filtration.UpsertPiezometerCountsRequest, userID int64) error
}

func Upsert(log *slog.Logger, upserter PiezometerCountsUpserter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.piezometer-counts.Upsert"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req filtration.UpsertPiezometerCountsRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)
			log.Error("failed to validate request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(validationErrors))
			return
		}

		if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
			log.Warn("access denied to organization", slog.Int64("org_id", req.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		if err := upserter.UpsertPiezometerCounts(r.Context(), req, userID); err != nil {
			log.Error("failed to upsert piezometer counts", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to save piezometer counts"))
			return
		}

		log.Info("piezometer counts upserted", slog.Int64("organization_id", req.OrganizationID))
		render.JSON(w, r, resp.OK())
	}
}
