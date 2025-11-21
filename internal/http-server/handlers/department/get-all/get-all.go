package get_all

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/department"
	"strconv"
)

// DepartmentGetter - интерфейс для получения списка
type DepartmentGetter interface {
	GetAllDepartments(ctx context.Context, orgID *int64) ([]*department.Model, error)
}

func New(log *slog.Logger, getter DepartmentGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.department.get_all.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Парсим (опциональный) фильтр organization_id
		var orgID *int64
		if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
			val, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter, must be a number"))
				return
			}
			orgID = &val
		}

		// 2. Вызываем метод репозитория
		departments, err := getter.GetAllDepartments(r.Context(), orgID)
		if err != nil {
			log.Error("failed to get all departments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve departments"))
			return
		}

		log.Info("successfully retrieved departments", slog.Int("count", len(departments)))
		render.JSON(w, r, departments)
	}
}
