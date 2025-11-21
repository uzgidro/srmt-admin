package visit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	OrganizationID  *int64  `json:"organization_id,omitempty"`
	VisitDate       *string `json:"visit_date,omitempty"`
	Description     *string `json:"description,omitempty"`
	ResponsibleName *string `json:"responsible_name,omitempty"`
}

type visitEditor interface {
	EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest) error
}

func Edit(log *slog.Logger, editor visitEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.visit.edit.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		storageReq := dto.EditVisitRequest{
			OrganizationID:  req.OrganizationID,
			Description:     req.Description,
			ResponsibleName: req.ResponsibleName,
		}

		// Parse visit_date if provided
		if req.VisitDate != nil {
			visitDate, err := time.Parse(time.RFC3339, *req.VisitDate)
			if err != nil {
				log.Warn("invalid visit_date format", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'visit_date' format, use ISO 8601 (e.g., 2024-01-15T10:30:00Z)"))
				return
			}
			storageReq.VisitDate = &visitDate
		}

		err = editor.EditVisit(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("visit not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Visit not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (org_id not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to update visit", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update visit"))
			return
		}

		log.Info("visit updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
