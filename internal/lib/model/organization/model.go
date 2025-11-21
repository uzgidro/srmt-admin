package organization

type Model struct {
	ID                     int64    `json:"id"`
	Name                   string   `json:"name"`
	ParentOrganizationID   *int64   `json:"parent_organization_id,omitempty"`
	ParentOrganizationName *string  `json:"parent_organization,omitempty"`
	Types                  []string `json:"types"`
	Items                  []*Model `json:"items,omitempty"`
}
