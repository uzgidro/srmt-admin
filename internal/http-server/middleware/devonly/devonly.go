package devonly

import "net/http"

// Guard returns a middleware that blocks requests when env is "prod".
// Non-prod environments pass through normally; in prod, requests get 404.
func Guard(env string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if env == "prod" {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
