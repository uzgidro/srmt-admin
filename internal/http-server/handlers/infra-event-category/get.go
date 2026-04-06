package infraeventcategory

import (
	"context"
	"log/slog"
	"net/http"

	infraeventcategorymodel "srmt-admin/internal/lib/model/infra-event-category"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
)

type categoryGetter interface {
	GetInfraEventCategories(ctx context.Context) ([]*infraeventcategorymodel.Model, error)
}

func Get(log *slog.Logger, getter categoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event-category.get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		categories, err := getter.GetInfraEventCategories(r.Context())
		if err != nil {
			log.Error("failed to get categories", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve categories"))
			return
		}

		if categories == nil {
			categories = make([]*infraeventcategorymodel.Model, 0)
		}
		render.JSON(w, r, categories)
	}
}
