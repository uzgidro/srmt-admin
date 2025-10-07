package auth

import (
	"net/http"
)

func AdminOnly(next http.Handler) http.Handler {
	return RequireAnyRole("admin")(next)
}

func SupremeOnly(next http.Handler) http.Handler {
	return RequireAnyRole("supreme")(next)

}
