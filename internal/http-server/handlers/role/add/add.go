package add

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
)

type Request struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

type Response struct {
	resp.Response
}

type RoleCreator interface {
	AddRole(ctx context.Context, name string, description string) (int64, error)
	GetUsersByRole(ctx context.Context, roleID int64) ([]user.Model, error)
	AssignRoleToUsers(ctx context.Context, roleID int64, userIDs []int64) error
}

func New(log *slog.Logger, roleCreator RoleCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.role.add.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		// Decode JSON
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		log.Info("request parsed", slog.Any("req", req))

		// Validate fields
		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)

			log.Error("failed to validate request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to validate request"))

			return
		}

		// add role
		id, err := roleCreator.AddRole(r.Context(), req.Name, req.Description)
		if err != nil {
			log.Info("failed to add role", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to add role"))
			return
		}

		// Get users with admin role
		users, err := roleCreator.GetUsersByRole(r.Context(), 2)
		if err != nil {
			log.Info("failed to get users by role", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			return
		}

		var userIDs []int64
		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}

		if err := roleCreator.AssignRoleToUsers(r.Context(), id, userIDs); err != nil {
			log.Info("failed to assign role to user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			return
		}

		log.Info("successfully added role", slog.Int64("id", id))

		render.Status(r, http.StatusCreated)
	}
}
