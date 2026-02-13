package access

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/access"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CardAllGetter interface {
	GetAllCards(ctx context.Context, filters dto.AccessCardFilters) ([]*access.AccessCard, error)
}

func GetCards(log *slog.Logger, svc CardAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetCards"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.AccessCardFilters

		if v := q.Get("employee_id"); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				filters.EmployeeID = &id
			}
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		cards, err := svc.GetAllCards(r.Context(), filters)
		if err != nil {
			log.Error("failed to get access cards", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access cards"))
			return
		}

		render.JSON(w, r, cards)
	}
}
