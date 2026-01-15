package dto

import "time"

// GetAllInstructionsFilters - Filters for querying instructions
type GetAllInstructionsFilters struct {
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

// AddInstructionRequest is the DTO for creating an instruction
type AddInstructionRequest struct {
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

// EditInstructionRequest is the DTO for updating an instruction
// All fields are pointers (optional) - only provided fields will be updated
type EditInstructionRequest struct {
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
