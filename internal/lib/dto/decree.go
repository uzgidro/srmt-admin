package dto

import "time"

// GetAllDecreesFilters - Filters for querying decrees
type GetAllDecreesFilters struct {
	TypeID               *int       `json:"type_id,omitempty"`
	StatusID             *int       `json:"status_id,omitempty"`
	OrganizationID       *int64     `json:"organization_id,omitempty"`
	ResponsibleContactID *int64     `json:"responsible_contact_id,omitempty"`
	ExecutorContactID    *int64     `json:"executor_contact_id,omitempty"`
	StartDate            *time.Time `json:"start_date,omitempty"`
	EndDate              *time.Time `json:"end_date,omitempty"`
	DueDateFrom          *time.Time `json:"due_date_from,omitempty"`
	DueDateTo            *time.Time `json:"due_date_to,omitempty"`
	NameSearch           *string    `json:"name_search,omitempty"`
	NumberSearch         *string    `json:"number_search,omitempty"`
}

// AddDecreeRequest is the DTO for creating a decree
type AddDecreeRequest struct {
	Name                 string                  `json:"name"`
	Number               *string                 `json:"number,omitempty"`
	DocumentDate         time.Time               `json:"document_date"`
	Description          *string                 `json:"description,omitempty"`
	TypeID               int                     `json:"type_id"`
	StatusID             *int                    `json:"status_id,omitempty"`
	ResponsibleContactID *int64                  `json:"responsible_contact_id,omitempty"`
	OrganizationID       *int64                  `json:"organization_id,omitempty"`
	ExecutorContactID    *int64                  `json:"executor_contact_id,omitempty"`
	DueDate              *time.Time              `json:"due_date,omitempty"`
	ParentDocumentID     *int64                  `json:"parent_document_id,omitempty"`
	FileIDs              []int64                 `json:"file_ids,omitempty"`
	LinkedDocuments      []LinkedDocumentRequest `json:"linked_documents,omitempty"`
}

// EditDecreeRequest is the DTO for updating a decree
// All fields are pointers (optional) - only provided fields will be updated
type EditDecreeRequest struct {
	Name                 *string                 `json:"name,omitempty"`
	Number               *string                 `json:"number,omitempty"`
	DocumentDate         *time.Time              `json:"document_date,omitempty"`
	Description          *string                 `json:"description,omitempty"`
	TypeID               *int                    `json:"type_id,omitempty"`
	StatusID             *int                    `json:"status_id,omitempty"`
	ResponsibleContactID *int64                  `json:"responsible_contact_id,omitempty"`
	OrganizationID       *int64                  `json:"organization_id,omitempty"`
	ExecutorContactID    *int64                  `json:"executor_contact_id,omitempty"`
	DueDate              *time.Time              `json:"due_date,omitempty"`
	ParentDocumentID     *int64                  `json:"parent_document_id,omitempty"`
	FileIDs              []int64                 `json:"file_ids,omitempty"`
	LinkedDocuments      []LinkedDocumentRequest `json:"linked_documents,omitempty"`
	StatusChangeComment  *string                 `json:"status_change_comment,omitempty"`
}

// ChangeStatusRequest is the DTO for changing document status
type ChangeStatusRequest struct {
	StatusID int     `json:"status_id" validate:"required,min=1"`
	Comment  *string `json:"comment,omitempty"`
}

// LinkedDocumentRequest is the DTO for linking documents
type LinkedDocumentRequest struct {
	LinkedDocumentType string  `json:"linked_document_type" validate:"required,oneof=decree report letter instruction legal_document"`
	LinkedDocumentID   int64   `json:"linked_document_id" validate:"required,min=1"`
	LinkDescription    *string `json:"link_description,omitempty"`
}
