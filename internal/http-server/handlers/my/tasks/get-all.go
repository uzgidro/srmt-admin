package tasks

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/dashboard"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type TasksGetter interface {
	GetHRMDashboardTasks(ctx context.Context, userID int64) ([]dashboard.Task, error)
}

func GetAll(log *slog.Logger, repo TasksGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.tasks.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		tasks, err := repo.GetHRMDashboardTasks(r.Context(), claims.ContactID)
		if err != nil {
			log.Error("failed to get tasks", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get tasks"))
			return
		}

		if tasks == nil {
			tasks = []dashboard.Task{}
		}
		render.JSON(w, r, tasks)
	}
}
