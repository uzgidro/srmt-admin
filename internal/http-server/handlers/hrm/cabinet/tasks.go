package cabinet

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// TaskRepository defines the interface for task operations
type TaskRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetPendingApprovalsForManager(ctx context.Context, managerID int64) ([]interface{}, error)
}

// GetMyTasks returns tasks for the currently authenticated employee
func GetMyTasks(log *slog.Logger, repo TaskRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyTasks"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Find employee by user_id
		employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee profile not found"))
				return
			}
			log.Error("failed to get employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee"))
			return
		}

		// Get pending tasks/approvals for this employee as manager
		// This returns vacation requests, document approvals, etc. that need this user's attention
		tasks, err := repo.GetPendingApprovalsForManager(r.Context(), employee.ID)
		if err != nil {
			log.Error("failed to get tasks", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve tasks"))
			return
		}

		// Convert to response format
		response := make([]hrm.MyTaskResponse, 0, len(tasks))
		for _, t := range tasks {
			// Type assertion to convert generic interface to task response
			// In a real implementation, this would properly handle different task types
			if taskMap, ok := t.(map[string]interface{}); ok {
				task := hrm.MyTaskResponse{}
				if id, ok := taskMap["id"].(int64); ok {
					task.ID = id
				}
				if title, ok := taskMap["title"].(string); ok {
					task.Title = title
				}
				if taskType, ok := taskMap["type"].(string); ok {
					task.TaskType = taskType
				}
				if status, ok := taskMap["status"].(string); ok {
					task.Status = status
				}
				task.Priority = "normal"
				response = append(response, task)
			}
		}

		log.Info("tasks retrieved", slog.Int64("employee_id", employee.ID), slog.Int("count", len(response)))
		render.JSON(w, r, response)
	}
}
