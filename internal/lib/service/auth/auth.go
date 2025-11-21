package auth

import (
	"context"
	"errors"
	// Импортируем ваш пакет middleware, чтобы получить доступ к функции извлечения claims
	mwauth "srmt-admin/internal/http-server/middleware/auth"
)

var ErrClaimsNotFound = errors.New("claims not found in context")

func GetUserID(ctx context.Context) (int64, error) {
	claims, ok := mwauth.ClaimsFromContext(ctx)

	if !ok || claims == nil {
		return 0, ErrClaimsNotFound
	}
	return claims.UserID, nil
}
