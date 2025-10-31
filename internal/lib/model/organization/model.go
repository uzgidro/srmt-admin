package organization

type Model struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	ParentOrganizationID *int64   `json:"parent_organization_id,omitempty"`
	Types                []string `json:"types"`
}
