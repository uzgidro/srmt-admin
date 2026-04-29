package solar

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/solar"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type ConfigUpserter interface {
	UpsertSolarConfig(ctx context.Context, req model.UpsertConfigRequest) error
}

func UpsertConfig(log *slog.Logger, repo ConfigUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.solar.UpsertConfig"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Defence-in-depth: route-level Tier 2 (sc/rais) is the primary gate,
		// but reject any non-admin caller here too in case wiring drifts.
		if !callerIsAdmin(r.Context()) {
			userID, _ := auth.GetUserID(r.Context())
			log.Warn("non-admin attempted solar config upsert",
				slog.Int64("user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("only sc/rais may modify config"))
			return
		}

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

		if err := repo.UpsertSolarConfig(r.Context(), req); err != nil {
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				log.Warn("CHECK violation", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid value (CHECK constraint)"))
				return
			}
			log.Error("failed to upsert solar config", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save config"))
			return
		}

		log.Info("solar config upserted", slog.Int64("organization_id", req.OrganizationID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
