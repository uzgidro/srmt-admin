package solar

import (
	"context"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	model "srmt-admin/internal/lib/model/solar"
)

// callerIsAdmin returns true iff the caller has sc or rais role.
// Used in handler-level defence-in-depth where a route-level Tier 2 gate
// already exists at the router (see router.go) but the handler also rejects
// to defend against future routing-mistakes that would expose POST/DELETE.
func callerIsAdmin(ctx context.Context) bool {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return false
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return true
		}
	}
	return false
}

// filterDailyDataForCaller restricts the response to records the caller is
// allowed to see. sc/rais see everything. Other roles (typically cascade) see
// only records for their own org. The handler MUST have already enforced
// claims.OrganizationID != 0 for non-admin roles before calling this.
//
// This is a defence-in-depth filter: even if the repo layer somehow returned
// records for foreign organizations (e.g. via a misused query-param hint that
// the handler accidentally trusted), this filter strips them before the JSON
// response leaves the process.
func filterDailyDataForCaller(ctx context.Context, list []model.DailyData) []model.DailyData {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return []model.DailyData{}
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return list
		}
	}
	out := make([]model.DailyData, 0, len(list))
	for _, rec := range list {
		if rec.OrganizationID == claims.OrganizationID {
			out = append(out, rec)
		}
	}
	return out
}
