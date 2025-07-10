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

// Authenticator создает middleware, которому для работы нужен наш JWT-сервис.
func Authenticator(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Извлекаем заголовок
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header required", http.StatusUnauthorized)
				return
			}

			// 2. Проверяем формат "Bearer <token>"
			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" {
				http.Error(w, "invalid auth header", http.StatusUnauthorized)
				return
			}

			tokenString := headerParts[1]

			// 3. Проверяем токен с помощью нашего сервиса
			claims, err := verifier.Verify(tokenString)
			if err != nil {
				// Здесь можно проверять тип ошибки (например, auth.ErrTokenExpired)
				// и отправлять более конкретный ответ.
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// 4. Помещаем ТИПИЗИРОВАННЫЕ claims в контекст
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext — это хелпер для безопасного извлечения claims в хендлерах.
func ClaimsFromContext(ctx context.Context) (*token.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*token.Claims)
	return claims, ok
}
