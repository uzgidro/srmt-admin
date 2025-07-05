package sign_up

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
)

type Request struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type Response struct {
	resp.Response
}

// UserCreator Construction must be equal to Storage method, or Service in future
type UserCreator interface {
	AddUser(ctx context.Context, name, passHash string) (int64, error)
}

func New(log *slog.Logger, userCreator UserCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.sign-up.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		// Decode JSON
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))

			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		log.Info("request parsed", slog.Any("req", req))

		// Validate fields
		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)

			log.Error("failed to validate request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validationErrors))

			return
		}

		// Hash password
		hashPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Error("failed to hashPassword password", sl.Err(err))

			render.JSON(w, r, resp.InternalServerError("failed to hashPassword password"))
			return
		}

		// save user
		id, err := userCreator.AddUser(r.Context(), req.Name, string(hashPassword))
		if err != nil {
			log.Info("failed to add user", sl.Err(err))

			render.JSON(w, r, resp.InternalServerError("failed to add user"))
			return
		}

		log.Info("successfully added user", slog.Int64("id", id))

		render.JSON(w, r, resp.Created())
	}
}
