package document

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DocumentRequestApprover interface {
	ApproveRequest(ctx context.Context, id int64) error
}

func ApproveRequest(log *slog.Logger, svc DocumentRequestApprover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.ApproveRequest"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		if err := svc.ApproveRequest(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrDocumentRequestNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document request not found"))
				return
			}
			log.Error("failed to approve document request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to approve document request"))
			return
		}

		render.JSON(w, r, resp.OK())
	}
}
