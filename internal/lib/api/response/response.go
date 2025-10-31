package response

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strings"
)

type Response struct {
	Status int    `json:"-"`
	Error  string `json:"error,omitempty"`
}

func Ok() Response {
	return Response{Status: http.StatusOK}
}

func Created() Response {
	return Response{
		Status: http.StatusCreated,
	}
}

func BadRequest(msg string) Response {
	return Response{
		Status: http.StatusBadRequest,
		Error:  msg,
	}
}

func Unauthorized(msg string) Response {
	return Response{
		Status: http.StatusUnauthorized,
		Error:  msg,
	}
}

func NotFound(msg string) Response {
	return Response{
		Status: http.StatusNotFound,
		Error:  msg,
	}
}

func InternalServerError(msg string) Response {
	return Response{
		Status: http.StatusInternalServerError,
		Error:  msg,
	}
}

func BadGateway(msg string) Response {
	return Response{
		Status: http.StatusBadGateway,
		Error:  msg,
	}
}

func ValidationErrors(errs validator.ValidationErrors) Response {
	var errMessages []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMessages = append(errMessages, fmt.Sprintf("field '%s' is required", err.Field()))
		case "min":
			errMessages = append(errMessages, fmt.Sprintf("field '%s' is too short", err.Field()))
		default:
			errMessages = append(errMessages, fmt.Sprintf("field '%s' is not valid", err.Field()))
		}
	}

	return Response{
		Status: http.StatusBadRequest,
		Error:  strings.Join(errMessages, "; "),
	}
}

func Conflict(msg string) Response {
	return Response{
		Status: http.StatusConflict,
		Error:  msg,
	}
}
