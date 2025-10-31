package organization_type

type OrganizationType struct {
	ID   string `bson:"_id,omitempty" json:"id"`
	Name string `bson:"name" json:"name"`
}
