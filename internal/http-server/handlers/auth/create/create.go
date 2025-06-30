package create

import (
	"golang.org/x/exp/slog"
	"net/http"
)

type Request struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// UserCreator Construction must be equal to Storage method, or Service in future
type UserCreator interface {
	AddUser(name, passHash string) (int64, error)
}

func New(log *slog.Logger, userCreator UserCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
