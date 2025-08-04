package auth

import (
	"net/http"
)

func AdminOnly(next http.Handler) http.Handler {
	return RequireAnyRole("admin")(next)
}
