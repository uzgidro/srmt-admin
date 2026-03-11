package piezometers

import (
	"net/http"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/token"
)

// mockTokenVerifier implements mwauth.TokenVerifier for injecting claims into context.
type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

// withAuth wraps a handler with the Authenticator middleware and a mock verifier
// that returns the given claims. The request must include "Authorization: Bearer test-token".
func withAuth(handler http.HandlerFunc, claims *token.Claims) http.Handler {
	verifier := &mockTokenVerifier{claims: claims}
	return mwauth.Authenticator(verifier)(handler)
}
