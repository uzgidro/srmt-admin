// Package cascadefilter restricts an organization list to the caller's
// cascade when the caller holds the "cascade" role. Other roles are
// passed through unchanged so existing behaviour is preserved.
package cascadefilter

import (
	"context"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/service/auth"
)

// Apply returns orgs filtered by cascade membership for the caller.
//
// Rules:
//   - No claims in ctx -> orgs unchanged.
//   - sc/rais role    -> orgs unchanged (full access).
//   - No cascade role -> orgs unchanged (other roles are not restricted here).
//   - cascade role:
//     - empty claims.OrganizationIDs -> empty slice (nothing visible).
//     - otherwise keep orgs whose ID is in claims.OrganizationIDs OR
//       whose ParentOrganizationID points at one of claims.OrganizationIDs.
//       Items (nested tree children) are preserved verbatim.
func Apply(ctx context.Context, orgs []*organization.Model) []*organization.Model {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return orgs
	}

	hasCascade := false
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return orgs
		}
		if role == "cascade" {
			hasCascade = true
		}
	}

	if !hasCascade {
		return orgs
	}

	if len(claims.OrganizationIDs) == 0 {
		return []*organization.Model{}
	}

	filtered := make([]*organization.Model, 0, len(orgs))
	for _, org := range orgs {
		if org == nil {
			continue
		}
		if auth.ContainsOrg(claims.OrganizationIDs, org.ID) {
			filtered = append(filtered, org)
			continue
		}
		if org.ParentOrganizationID != nil && auth.ContainsOrg(claims.OrganizationIDs, *org.ParentOrganizationID) {
			filtered = append(filtered, org)
		}
	}
	return filtered
}
