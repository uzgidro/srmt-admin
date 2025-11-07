package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"strconv"
)

type UserGetter interface {
	GetAllUsers(ctx context.Context, filters dto.GetAllUsersFilters) ([]*user.Model, error)
}

type ResponseUser struct {
	ID    int64    `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

func New(log *slog.Logger, userGetter UserGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.users.get.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// --- (ИСПРАВЛЕНО) 1. Парсим фильтры из query-параметров ---
		var filters dto.GetAllUsersFilters
		q := r.URL.Query()

		if orgIDStr := q.Get("organization_id"); orgIDStr != "" {
			val, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
				return
			}
			filters.OrganizationID = &val
		}

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'department_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'department_id' parameter"))
				return
			}
			filters.DepartmentID = &val
		}

		if isActiveStr := q.Get("is_active"); isActiveStr != "" {
			val, err := strconv.ParseBool(isActiveStr)
			if err != nil {
				log.Warn("invalid 'is_active' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'is_active' parameter"))
				return
			}
			filters.IsActive = &val
		}
		// ---

		// --- (ИСПРАВЛЕНО) 2. Вызываем репозиторий с фильтрами ---
		users, err := userGetter.GetAllUsers(r.Context(), filters)
		if err != nil {
			log.Error("failed to get all users", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve users"))
			return
		}

		// 3. Преобразуем полные модели (*user.Model) в DTO для ответа (ResponseUser)
		responseUsers := make([]ResponseUser, len(users))
		for i, u := range users {
			responseUsers[i] = ResponseUser{
				ID:    u.ID,
				Name:  u.FIO, // (Маппим FIO в Name, как ты и делал)
				Roles: u.Roles,
			}
		}

		log.Info("successfully retrieved all users", slog.Int("count", len(responseUsers)))
		render.JSON(w, r, responseUsers)
	}
}
