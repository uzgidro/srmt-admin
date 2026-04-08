package investments

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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	Name     *string  `json:"name,omitempty"`
	TypeID   *int     `json:"type_id,omitempty"`
	StatusID *int     `json:"status_id,omitempty"`
	Cost     *float64 `json:"cost,omitempty"`
	Comments *string  `json:"comments,omitempty"`
	FileIDs  []int64  `json:"file_ids,omitempty"`
}

type investmentEditor interface {
	EditInvestment(ctx context.Context, id int64, req dto.EditInvestmentRequest) error
	UnlinkInvestmentFiles(ctx context.Context, investmentID int64) error
	LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error
}

func Edit(log *slog.Logger, editor investmentEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.edit"
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

		// Build storage request
		storageReq := dto.EditInvestmentRequest{
			Name:     req.Name,
			TypeID:   req.TypeID,
			StatusID: req.StatusID,
			Cost:     req.Cost,
			Comments: req.Comments,
			FileIDs:  req.FileIDs,
		}

		// Update investment
		err = editor.EditInvestment(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("investment not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Investment not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (invalid status_id)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status ID"))
				return
			}
			log.Error("failed to update investment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update investment"))
			return
		}

		// Update file links if explicitly requested
		if req.FileIDs != nil {
			// Remove old links
			if err := editor.UnlinkInvestmentFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(req.FileIDs) > 0 {
				if err := editor.LinkInvestmentFiles(r.Context(), id, req.FileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("investment updated successfully",
			slog.Int64("id", id),
			slog.Bool("files_updated", req.FileIDs != nil),
			slog.Int("total_files", len(req.FileIDs)),
		)

		render.JSON(w, r, resp.OK())
	}
}
