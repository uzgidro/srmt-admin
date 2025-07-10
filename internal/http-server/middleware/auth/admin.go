package auth

import (
	"context"
	"net/http"
	"srmt-admin/internal/token"
)

// AdminOnly — это middleware, который пропускает только администраторов.
// Он должен быть использован ПОСЛЕ middleware аутентификации.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// IsAdmin — ваша функция для проверки роли в контексте
		isAdmin := isAdmin(r.Context())
		if !isAdmin {
			// Если не админ, возвращаем 403 Forbidden и прерываем цепочку.
			http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
			return
		}
		// Если админ, передаем управление следующему хендлеру в цепочке.
		next.ServeHTTP(w, r)
	})
}

func isAdmin(ctx context.Context) bool {
	claims, ok := ctx.Value(claimsKey).(*token.Claims)
	if ok {
		for _, item := range claims.Roles {
			if item == "admin" {
				return true
			}
		}
		return false
	}
	return false
}
