package documents

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/profile"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DocumentsGetter interface {
	GetMyDocuments(ctx context.Context, employeeID int64) ([]*profile.MyDocument, error)
}

func GetAll(log *slog.Logger, repo DocumentsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.documents.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		docs, err := repo.GetMyDocuments(r.Context(), claims.ContactID)
		if err != nil {
			log.Error("failed to get documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get documents"))
			return
		}

		if docs == nil {
			docs = []*profile.MyDocument{}
		}
		render.JSON(w, r, docs)
	}
}
