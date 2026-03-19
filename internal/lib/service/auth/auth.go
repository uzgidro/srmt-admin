package auth

import (
	"context"
	"errors"
	// Импортируем ваш пакет middleware, чтобы получить доступ к функции извлечения claims
	mwauth "srmt-admin/internal/http-server/middleware/auth"
)

var (
	ErrClaimsNotFound  = errors.New("claims not found in context")
	ErrForbidden       = errors.New("access denied")
	ErrNoOrganization  = errors.New("user has no organization assigned")
)

func GetUserID(ctx context.Context) (int64, error) {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return 0, ErrClaimsNotFound
	}
	return claims.UserID, nil
}

func GetOrganizationID(ctx context.Context) (int64, error) {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return 0, ErrClaimsNotFound
	}
	return claims.OrganizationID, nil
}

// CheckOrgAccessBatch checks access for multiple organization IDs.
// Returns the first error encountered, skipping duplicate IDs.
func CheckOrgAccessBatch(ctx context.Context, orgIDs []int64) error {
	seen := make(map[int64]struct{})
	for _, id := range orgIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if err := CheckOrgAccess(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// CheckOrgAccess returns nil if the user has access to the given organization.
// sc/rais roles have full access; reservoir role is limited to own org only.
func CheckOrgAccess(ctx context.Context, resourceOrgID int64) error {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return ErrClaimsNotFound
	}

	if resourceOrgID == 0 {
		return ErrNoOrganization
	}

	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return nil
		}
	}

	if claims.OrganizationID == 0 {
		return ErrNoOrganization
	}
	if claims.OrganizationID != resourceOrgID {
		return ErrForbidden
	}
	return nil
}
