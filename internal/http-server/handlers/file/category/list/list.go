package list

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/category"
)

type CategoryGetter interface {
	GetAllCategories(ctx context.Context) ([]category.Model, error)
}

func New(log *slog.Logger, getter CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.category.list.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		categories, err := getter.GetAllCategories(r.Context())
		if err != nil {
			log.Error("failed to get all categories", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve categories"))
			return
		}

		log.Info("successfully retrieved all categories", slog.Int("count", len(categories)))

		render.JSON(w, r, categories)
	}
}
