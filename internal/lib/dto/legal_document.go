package dto

import "time"

// GetAllLegalDocumentsFilters - Filters for querying legal documents
type GetAllLegalDocumentsFilters struct {
	TypeID       *int       `json:"type_id,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	NameSearch   *string    `json:"name_search,omitempty"`
	NumberSearch *string    `json:"number_search,omitempty"`
}

// AddLegalDocumentRequest is the DTO for creating a legal document
type AddLegalDocumentRequest struct {
	Name         string    `json:"name"`
	Number       *string   `json:"number,omitempty"`
	DocumentDate time.Time `json:"document_date"`
	TypeID       int       `json:"type_id"`
	FileIDs      []int64   `json:"file_ids,omitempty"`
}

// EditLegalDocumentRequest is the DTO for updating a legal document
// All fields are pointers (optional) - only provided fields will be updated
type EditLegalDocumentRequest struct {
	Name         *string    `json:"name,omitempty"`
	Number       *string    `json:"number,omitempty"`
	DocumentDate *time.Time `json:"document_date,omitempty"`
	TypeID       *int       `json:"type_id,omitempty"`
	FileIDs      []int64    `json:"file_ids,omitempty"`
}
