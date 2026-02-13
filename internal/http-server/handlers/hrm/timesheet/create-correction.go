package timesheet

import (
	"context"
	"errors"
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

type CorrectionCreator interface {
	CreateCorrection(ctx context.Context, req dto.CreateTimesheetCorrectionRequest, requestedBy int64) (int64, error)
}

type CreateCorrectionResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

func CreateCorrection(log *slog.Logger, svc CorrectionCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.CreateCorrection"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		var req dto.CreateTimesheetCorrectionRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := svc.CreateCorrection(r.Context(), req, claims.ContactID)
		if err != nil {
			log.Error("failed to create correction", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create correction"))
			return
		}

		log.Info("correction created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, CreateCorrectionResponse{Response: resp.OK(), ID: id})
	}
}
