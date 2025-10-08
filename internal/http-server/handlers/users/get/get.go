package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
)

type UserGetter interface {
	GetAllUsers(ctx context.Context) ([]user.Model, error)
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

		users, err := userGetter.GetAllUsers(r.Context())
		if err != nil {
			log.Error("failed to get all users", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve users"))
			return
		}

		// Преобразуем модели из БД в модели для ответа
		responseUsers := make([]ResponseUser, len(users))
		for i, u := range users {
			responseUsers[i] = ResponseUser{
				ID:    u.ID,
				Name:  u.Name,
				Roles: u.Roles,
			}
		}

		log.Info("successfully retrieved all users", slog.Int("count", len(responseUsers)))

		render.JSON(w, r, responseUsers)
	}
}
