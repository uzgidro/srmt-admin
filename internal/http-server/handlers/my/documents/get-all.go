package documents

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/personnel"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DocumentsService interface {
	GetByEmployeeID(ctx context.Context, employeeID int64) (*personnel.Record, error)
	GetDocuments(ctx context.Context, recordID int64) ([]*personnel.Document, error)
}

func GetAll(log *slog.Logger, svc DocumentsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.documents.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		record, err := svc.GetByEmployeeID(r.Context(), claims.ContactID)
		if err != nil {
			if errors.Is(err, storage.ErrPersonnelRecordNotFound) {
				render.JSON(w, r, []*personnel.Document{})
				return
			}
			log.Error("failed to get personnel record", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get documents"))
			return
		}

		docs, err := svc.GetDocuments(r.Context(), record.ID)
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
