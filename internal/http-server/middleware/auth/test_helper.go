package auth

import (
	"context"
	"srmt-admin/internal/token"
)

// ContextWithClaims is a test helper that adds claims to a context
// This should only be used in tests to simulate authenticated requests
func ContextWithClaims(ctx context.Context, claims *token.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}
