package training

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type DevelopmentPlanCreator interface {
	CreateDevelopmentPlan(ctx context.Context, req dto.CreateDevelopmentPlanRequest, createdBy int64) (int64, error)
}

func CreateDevelopmentPlan(log *slog.Logger, svc DevelopmentPlanCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.CreateDevelopmentPlan"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		var req dto.CreateDevelopmentPlanRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request body"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(err.Error()))
			return
		}

		id, err := svc.CreateDevelopmentPlan(r.Context(), req, claims.ContactID)
		if err != nil {
			log.Error("failed to create development plan", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create development plan"))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]int64{"id": id})
	}
}
