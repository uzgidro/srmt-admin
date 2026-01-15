package decrees

import (
	"context"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	decree_type "srmt-admin/internal/lib/model/decree-type"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type decreeTypeGetter interface {
	GetAllDecreeTypes(ctx context.Context) ([]decree_type.Model, error)
}

func GetTypes(log *slog.Logger, getter decreeTypeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.decrees.get-types"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		types, err := getter.GetAllDecreeTypes(r.Context())
		if err != nil {
			log.Error("failed to get decree types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve decree types"))
			return
		}

		log.Info("successfully retrieved decree types", slog.Int("count", len(types)))
		render.JSON(w, r, types)
	}
}
