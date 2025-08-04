package auth

import (
	"context"
	"net/http"
	"srmt-admin/internal/token"
	"strings"
)

type contextKey string

const claimsKey = contextKey("claims")

type TokenVerifier interface {
	Verify(token string) (*token.Claims, error)
}

func Authenticator(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header required", http.StatusUnauthorized)
				return
			}

			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" {
				http.Error(w, "invalid auth header", http.StatusUnauthorized)
				return
			}

			tokenString := headerParts[1]

			claims, err := verifier.Verify(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*token.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*token.Claims)
	return claims, ok
}

func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !hasAnyRole(r.Context(), roles...) {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasAnyRole(ctx context.Context, requiredRoles ...string) bool {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return false
	}

	userRoles := make(map[string]struct{}, len(claims.Roles))
	for _, role := range claims.Roles {
		userRoles[role] = struct{}{}
	}

	for _, requiredRole := range requiredRoles {
		if _, found := userRoles[requiredRole]; found {
			return true
		}
	}

	return false
}
