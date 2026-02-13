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

type DocumentDeleter interface {
	DeleteDocument(ctx context.Context, id int64) error
}

func DeleteDocument(log *slog.Logger, svc DocumentDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.DeleteDocument"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		if err := svc.DeleteDocument(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrHRDocumentNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document not found"))
				return
			}
			if errors.Is(err, storage.ErrInvalidStatus) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Can only delete documents in draft status"))
				return
			}
			log.Error("failed to delete document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete document"))
			return
		}

		render.JSON(w, r, resp.OK())
	}
}
