package sc

import (
	"slices"

	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/visit"
)

// sortOrgIDs sorts organization IDs depth-first by parent hierarchy.
// Parent orgs come first, immediately followed by their children sorted by ID.
// Root orgs (no parent) are sorted by ID among themselves.
func sortOrgIDs(orgIDs []int64, parentMap map[int64]*int64) []int64 {
	if len(orgIDs) == 0 {
		return orgIDs
	}

	// Build tree: parentID -> sorted children
	children := make(map[int64][]int64) // parentID -> child orgIDs
	var roots []int64

	for _, id := range orgIDs {
		parentPtr, exists := parentMap[id]
		if !exists || parentPtr == nil {
			roots = append(roots, id)
		} else {
			children[*parentPtr] = append(children[*parentPtr], id)
		}
	}

	slices.Sort(roots)
	for k := range children {
		slices.Sort(children[k])
	}

	// Depth-first walk
	orgIDSet := make(map[int64]bool, len(orgIDs))
	for _, id := range orgIDs {
		orgIDSet[id] = true
	}

	result := make([]int64, 0, len(orgIDs))

	var walk func(id int64)
	walk = func(id int64) {
		if orgIDSet[id] {
			result = append(result, id)
		}
		for _, child := range children[id] {
			walk(child)
		}
	}

	// Walk from roots
	for _, root := range roots {
		walk(root)
	}

	// Handle orphans: children whose parent is not in orgIDs and not a root
	// Handle orphans: children whose parent is not in orgIDs (phantom parent)
	var orphanParents []int64
	for parentID := range children {
		if !orgIDSet[parentID] {
			orphanParents = append(orphanParents, parentID)
		}
	}
	slices.Sort(orphanParents)

	for _, parentID := range orphanParents {
		for _, child := range children[parentID] {
			result = append(result, child)
		}
	}

	return result
}

// determineOrgType determines the organization type from a list of types.
// Priority: micro > mini > ges (more specific wins).
func determineOrgType(types []string) string {
	for _, t := range types {
		if t == "micro" {
			return "micro"
		}
	}
	for _, t := range types {
		if t == "mini" {
			return "mini"
		}
	}
	for _, t := range types {
		if t == "ges" {
			return "ges"
		}
	}
	return ""
}

// sortVisitsByOrg sorts visits by parent_id → org_id of their organization.
func sortVisitsByOrg(visits []*visit.ResponseModel, parentMap map[int64]*int64) {
	orgIDs := make([]int64, 0)
	seen := make(map[int64]bool)
	for _, v := range visits {
		if !seen[v.OrganizationID] {
			orgIDs = append(orgIDs, v.OrganizationID)
			seen[v.OrganizationID] = true
		}
	}
	sorted := sortOrgIDs(orgIDs, parentMap)
	orderMap := make(map[int64]int, len(sorted))
	for i, id := range sorted {
		orderMap[id] = i
	}
	slices.SortStableFunc(visits, func(a, b *visit.ResponseModel) int {
		return orderMap[a.OrganizationID] - orderMap[b.OrganizationID]
	})
}

// sortIncidentsByOrg sorts incidents: NULL org_id first, then by parent_id → org_id.
func sortIncidentsByOrg(incidents []*incident.ResponseModel, parentMap map[int64]*int64) {
	orgIDs := make([]int64, 0)
	seen := make(map[int64]bool)
	for _, inc := range incidents {
		if inc.OrganizationID != nil && !seen[*inc.OrganizationID] {
			orgIDs = append(orgIDs, *inc.OrganizationID)
			seen[*inc.OrganizationID] = true
		}
	}
	sorted := sortOrgIDs(orgIDs, parentMap)
	orderMap := make(map[int64]int, len(sorted))
	for i, id := range sorted {
		orderMap[id] = i + 1 // +1 so NULL (0) comes first
	}
	slices.SortStableFunc(incidents, func(a, b *incident.ResponseModel) int {
		aOrder := 0
		bOrder := 0
		if a.OrganizationID != nil {
			aOrder = orderMap[*a.OrganizationID]
		}
		if b.OrganizationID != nil {
			bOrder = orderMap[*b.OrganizationID]
		}
		return aOrder - bOrder
	})
}
