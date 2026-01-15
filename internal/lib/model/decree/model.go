package decree

import (
	"time"

	"srmt-admin/internal/lib/dto"
	decree_type "srmt-admin/internal/lib/model/decree-type"
	document_status "srmt-admin/internal/lib/model/document-status"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
)

// ContactShortInfo represents minimal contact information
type ContactShortInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// OrganizationShortInfo represents minimal organization information
type OrganizationShortInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ParentDecreeInfo represents minimal parent decree information
type ParentDecreeInfo struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Number       *string    `json:"number,omitempty"`
	DocumentDate *time.Time `json:"document_date,omitempty"`
}

// DocumentLink represents a link to another document
type DocumentLink struct {
	ID              int64   `json:"id"`
	DocumentType    string  `json:"document_type"`
	DocumentID      int64   `json:"document_id"`
	DocumentName    string  `json:"document_name"`
	DocumentNumber  *string `json:"document_number,omitempty"`
	LinkDescription *string `json:"link_description,omitempty"`
}

// Model is the internal DB representation of a decree
type Model struct {
	ID                   int64      `json:"id"`
	Name                 string     `json:"name"`
	Number               *string    `json:"number,omitempty"`
	DocumentDate         time.Time  `json:"document_date"`
	Description          *string    `json:"description,omitempty"`
	TypeID               int        `json:"type_id"`
	StatusID             int        `json:"status_id"`
	ResponsibleContactID *int64     `json:"responsible_contact_id,omitempty"`
	OrganizationID       *int64     `json:"organization_id,omitempty"`
	ExecutorContactID    *int64     `json:"executor_contact_id,omitempty"`
	DueDate              *time.Time `json:"due_date,omitempty"`
	ParentDocumentID     *int64     `json:"parent_document_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	CreatedByUserID      *int64     `json:"created_by_user_id,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
	UpdatedByUserID      *int64     `json:"updated_by_user_id,omitempty"`
}

// ResponseModel includes joined data and files
type ResponseModel struct {
	ID                 int64                      `json:"id"`
	Name               string                     `json:"name"`
	Number             *string                    `json:"number,omitempty"`
	DocumentDate       time.Time                  `json:"document_date"`
	Description        *string                    `json:"description,omitempty"`
	Type               decree_type.Model          `json:"type"`
	Status             document_status.ShortModel `json:"status"`
	ResponsibleContact *ContactShortInfo          `json:"responsible_contact,omitempty"`
	Organization       *OrganizationShortInfo     `json:"organization,omitempty"`
	ExecutorContact    *ContactShortInfo          `json:"executor_contact,omitempty"`
	DueDate            *time.Time                 `json:"due_date,omitempty"`
	ParentDocument     *ParentDecreeInfo          `json:"parent_document,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	CreatedBy          *user.ShortInfo            `json:"created_by,omitempty"`
	UpdatedAt          *time.Time                 `json:"updated_at,omitempty"`
	UpdatedBy          *user.ShortInfo            `json:"updated_by,omitempty"`
	Files              []file.Model               `json:"files,omitempty"`
	LinkedDocuments    []DocumentLink             `json:"linked_documents,omitempty"`
}

// ResponseWithURLs is the API response with presigned file URLs
type ResponseWithURLs struct {
	ID                 int64                      `json:"id"`
	Name               string                     `json:"name"`
	Number             *string                    `json:"number,omitempty"`
	DocumentDate       time.Time                  `json:"document_date"`
	Description        *string                    `json:"description,omitempty"`
	Type               decree_type.Model          `json:"type"`
	Status             document_status.ShortModel `json:"status"`
	ResponsibleContact *ContactShortInfo          `json:"responsible_contact,omitempty"`
	Organization       *OrganizationShortInfo     `json:"organization,omitempty"`
	ExecutorContact    *ContactShortInfo          `json:"executor_contact,omitempty"`
	DueDate            *time.Time                 `json:"due_date,omitempty"`
	ParentDocument     *ParentDecreeInfo          `json:"parent_document,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	CreatedBy          *user.ShortInfo            `json:"created_by,omitempty"`
	UpdatedAt          *time.Time                 `json:"updated_at,omitempty"`
	UpdatedBy          *user.ShortInfo            `json:"updated_by,omitempty"`
	Files              []dto.FileResponse         `json:"files,omitempty"`
	LinkedDocuments    []DocumentLink             `json:"linked_documents,omitempty"`
}

// StatusHistory represents status change history
type StatusHistory struct {
	ID        int64                       `json:"id"`
	From      *document_status.ShortModel `json:"from_status,omitempty"`
	To        document_status.ShortModel  `json:"to_status"`
	ChangedAt time.Time                   `json:"changed_at"`
	ChangedBy *user.ShortInfo             `json:"changed_by,omitempty"`
	Comment   *string                     `json:"comment,omitempty"`
}

// ShortInfo represents minimal decree information for references
type ShortInfo struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Number *string `json:"number,omitempty"`
}
