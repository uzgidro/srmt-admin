package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Detail is one structured violation entry inside an error response. Free-form
// shape so handlers can include whatever context the frontend needs to render
// a precise message — e.g. {"organization_id": 16, "field": "consumption_m3_s",
// "value": -1.5}. Frontend keys off the top-level `code` to know which fields
// to expect.
type Detail map[string]any

// Response is the generic envelope for both success and error JSON bodies.
//
// `Code` and `Details` are recent additions to support structured errors that
// the frontend can localize and bind to specific fields/rows. Both are
// omitempty — legacy callers using BadRequest/Forbidden/etc. produce the same
// wire shape as before (`{"error": "..."}`), so the change is backwards
// compatible.
type Response struct {
	Status  int      `json:"-"`
	Error   string   `json:"error,omitempty"`
	Code    string   `json:"code,omitempty"`
	Details []Detail `json:"details,omitempty"`
}

func OK() Response {
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

// BadRequestStructured returns a 400 with a stable machine-readable code and
// per-violation details. Use this when the frontend needs to bind the error
// to a specific field/row (e.g. validation failures, business-rule
// violations). For free-form messages with no structure, use BadRequest.
func BadRequestStructured(code, msg string, details []Detail) Response {
	return Response{
		Status:  http.StatusBadRequest,
		Error:   msg,
		Code:    code,
		Details: details,
	}
}

func Forbidden(msg string) Response {
	return Response{
		Status: http.StatusForbidden,
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

func Delete() Response {
	return Response{
		Status: http.StatusNoContent,
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
