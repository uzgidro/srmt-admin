package reports

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type changeStatusRequest struct {
	StatusID int     `json:"status_id" validate:"required,min=1"`
	Comment  *string `json:"comment,omitempty"`
}

type reportStatusChanger interface {
	EditReport(ctx context.Context, id int64, req dto.EditReportRequest, updatedByID int64) error
	AddReportStatusHistoryComment(ctx context.Context, reportID int64, comment string) error
}

func ChangeStatus(log *slog.Logger, changer reportStatusChanger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reports.change-status"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req changeStatusRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		editReq := dto.EditReportRequest{
			StatusID: &req.StatusID,
		}

		err = changer.EditReport(r.Context(), id, editReq, userID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("report not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Report not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("invalid status_id", slog.Int("status_id", req.StatusID))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status_id"))
				return
			}
			log.Error("failed to change report status", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to change status"))
			return
		}

		// Add comment to the latest status history entry if provided
		if req.Comment != nil && *req.Comment != "" {
			if err := changer.AddReportStatusHistoryComment(r.Context(), id, *req.Comment); err != nil {
				log.Warn("failed to add status change comment", sl.Err(err))
			}
		}

		log.Info("report status changed successfully", slog.Int64("id", id), slog.Int("new_status_id", req.StatusID))
		render.JSON(w, r, resp.OK())
	}
}
