package api_key

import (
	"net/http"
)

func RequireAPIKey(validKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedKey := r.Header.Get("X-API-Key")
			if providedKey != validKey {
				http.Error(w, "Forbidden: invalid API key", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
