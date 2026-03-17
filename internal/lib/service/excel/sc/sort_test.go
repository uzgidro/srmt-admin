package sc

import (
	"slices"
	"testing"

	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/visit"
)

func TestSortOrgIDs(t *testing.T) {
	tests := []struct {
		name      string
		orgIDs    []int64
		parentMap map[int64]*int64
		expected  []int64
	}{
		{
			name:      "empty input",
			orgIDs:    []int64{},
			parentMap: map[int64]*int64{},
			expected:  []int64{},
		},
		{
			name:      "single org no parent",
			orgIDs:    []int64{5},
			parentMap: map[int64]*int64{5: nil},
			expected:  []int64{5},
		},
		{
			name:   "flat orgs sorted by id",
			orgIDs: []int64{12, 5, 23},
			parentMap: map[int64]*int64{
				5:  nil,
				12: nil,
				23: nil,
			},
			expected: []int64{5, 12, 23},
		},
		{
			name:   "children grouped after parent depth-first",
			orgIDs: []int64{10, 5, 12, 3, 11},
			parentMap: map[int64]*int64{
				3:  nil,
				5:  nil,
				10: ptr(int64(3)),
				11: ptr(int64(3)),
				12: ptr(int64(5)),
			},
			// cascade 3, then children 10, 11, then cascade 5, then child 12
			expected: []int64{3, 10, 11, 5, 12},
		},
		{
			name:   "children without parent in list still sorted",
			orgIDs: []int64{10, 11},
			parentMap: map[int64]*int64{
				10: ptr(int64(3)),
				11: ptr(int64(3)),
			},
			// parent 3 not in orgIDs, children grouped under virtual parent 3
			expected: []int64{10, 11},
		},
		{
			name:   "org missing from parentMap treated as root",
			orgIDs: []int64{99, 5},
			parentMap: map[int64]*int64{
				5: nil,
			},
			expected: []int64{5, 99},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortOrgIDs(tt.orgIDs, tt.parentMap)
			if !slices.Equal(result, tt.expected) {
				t.Errorf("sortOrgIDs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestSortVisitsByOrg(t *testing.T) {
	parentMap := map[int64]*int64{
		3:  nil,
		5:  nil,
		10: ptr(int64(3)),
	}

	visits := []*visit.ResponseModel{
		{ID: 1, OrganizationID: 5, OrganizationName: "Org5"},
		{ID: 2, OrganizationID: 10, OrganizationName: "Org10"},
		{ID: 3, OrganizationID: 3, OrganizationName: "Org3"},
	}

	sortVisitsByOrg(visits, parentMap)

	expectedOrgOrder := []int64{3, 10, 5}
	for i, v := range visits {
		if v.OrganizationID != expectedOrgOrder[i] {
			t.Errorf("position %d: got org_id=%d, want %d", i, v.OrganizationID, expectedOrgOrder[i])
		}
	}
}

func TestSortIncidentsByOrg_NullFirst(t *testing.T) {
	parentMap := map[int64]*int64{
		5:  nil,
		10: ptr(int64(3)),
		3:  nil,
	}

	orgID5 := int64(5)
	orgID10 := int64(10)

	incidents := []*incident.ResponseModel{
		{ID: 1, OrganizationID: &orgID5},
		{ID: 2, OrganizationID: nil},
		{ID: 3, OrganizationID: &orgID10},
	}

	sortIncidentsByOrg(incidents, parentMap)

	if incidents[0].ID != 2 {
		t.Errorf("expected NULL org incident first, got ID=%d", incidents[0].ID)
	}
	if incidents[1].OrganizationID == nil || *incidents[1].OrganizationID != 5 {
		t.Errorf("expected org 5 second, got %v", incidents[1].OrganizationID)
	}
	if incidents[2].OrganizationID == nil || *incidents[2].OrganizationID != 10 {
		t.Errorf("expected org 10 third, got %v", incidents[2].OrganizationID)
	}
}
