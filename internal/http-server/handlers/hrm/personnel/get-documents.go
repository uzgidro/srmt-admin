package personnel

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/personnel"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DocumentsGetter interface {
	GetDocuments(ctx context.Context, recordID int64) ([]*personnel.Document, error)
}

func GetDocuments(log *slog.Logger, svc DocumentsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.personnel.GetDocuments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		docs, err := svc.GetDocuments(r.Context(), id)
		if err != nil {
			log.Error("failed to get documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get documents"))
			return
		}

		if docs == nil {
			docs = []*personnel.Document{}
		}
		render.JSON(w, r, docs)
	}
}
