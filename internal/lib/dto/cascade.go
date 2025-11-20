package dto

import "srmt-admin/internal/lib/model/contact"

// CascadeWithDetails represents a cascade organization with contacts and discharge information
type CascadeWithDetails struct {
	ID                     int64                 `json:"id"`
	Name                   string                `json:"name"`
	ParentOrganizationID   *int64                `json:"parent_organization_id,omitempty"`
	ParentOrganizationName *string               `json:"parent_organization,omitempty"`
	Types                  []string              `json:"types"`
	Contacts               []*contact.Model      `json:"contacts"`
	CurrentDischarge       *float64              `json:"current_discharge,omitempty"` // Текущий расход воды в м³/с
	Items                  []*CascadeWithDetails `json:"items,omitempty"`             // Nested organizations (HPPs for cascades)
}
