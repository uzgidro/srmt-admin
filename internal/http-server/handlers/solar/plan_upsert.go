package solar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/solar"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type PlanUpserter interface {
	BulkUpsertSolarPlan(ctx context.Context, plans []model.UpsertPlanRequest, userID int64) error
}

func BulkUpsertPlan(log *slog.Logger, repo PlanUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.solar.BulkUpsertPlan"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Defence-in-depth: route-level Tier 2 (sc/rais) is the primary gate,
		// but reject any non-admin caller here too in case wiring drifts.
		if !callerIsAdmin(r.Context()) {
			userID, _ := auth.GetUserID(r.Context())
			log.Warn("non-admin attempted solar plan upsert",
				slog.Int64("user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("only sc/rais may modify plans"))
			return
		}

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Warn("no user id in context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var req model.BulkUpsertPlanRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request format"))
			return
		}

		// Struct-level validation cascades into each plan via the `dive` tag.
		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Defence-in-depth: validator's `gte=0` already covers this, but a
		// belt-and-braces explicit check guards against tag drift.
		for i, p := range req.Plans {
			if p.PlanThousandKWh < 0 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]any{
					"error":      fmt.Sprintf("plan_thousand_kwh must be >= 0 for organization_id=%d", p.OrganizationID),
					"item_index": i,
				})
				return
			}
		}

		if err := repo.BulkUpsertSolarPlan(r.Context(), req.Plans, userID); err != nil {
			log.Error("failed to upsert solar plans", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save plans"))
			return
		}

		log.Info("solar plans upserted", slog.Int("count", len(req.Plans)), slog.Int64("user_id", userID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
