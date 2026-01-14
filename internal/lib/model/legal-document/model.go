package legal_document

import (
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	legal_document_type "srmt-admin/internal/lib/model/legal-document-type"
	"srmt-admin/internal/lib/model/user"
	"time"
)

// Model is the internal DB representation of a legal document
type Model struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Number          *string    `json:"number,omitempty"`
	DocumentDate    time.Time  `json:"document_date"`
	TypeID          int        `json:"type_id"`
	CreatedAt       time.Time  `json:"created_at"`
	CreatedByUserID *int64     `json:"created_by_user_id,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	UpdatedByUserID *int64     `json:"updated_by_user_id,omitempty"`
}

// ResponseModel includes joined data and files
type ResponseModel struct {
	ID           int64                     `json:"id"`
	Name         string                    `json:"name"`
	Number       *string                   `json:"number,omitempty"`
	DocumentDate time.Time                 `json:"document_date"`
	Type         legal_document_type.Model `json:"type"`
	CreatedAt    time.Time                 `json:"created_at"`
	CreatedBy    *user.ShortInfo           `json:"created_by,omitempty"`
	UpdatedAt    *time.Time                `json:"updated_at,omitempty"`
	UpdatedBy    *user.ShortInfo           `json:"updated_by,omitempty"`
	Files        []file.Model              `json:"files,omitempty"`
}

// ResponseWithURLs is the API response with presigned file URLs
type ResponseWithURLs struct {
	ID           int64                     `json:"id"`
	Name         string                    `json:"name"`
	Number       *string                   `json:"number,omitempty"`
	DocumentDate time.Time                 `json:"document_date"`
	Type         legal_document_type.Model `json:"type"`
	CreatedAt    time.Time                 `json:"created_at"`
	CreatedBy    *user.ShortInfo           `json:"created_by,omitempty"`
	UpdatedAt    *time.Time                `json:"updated_at,omitempty"`
	UpdatedBy    *user.ShortInfo           `json:"updated_by,omitempty"`
	Files        []dto.FileResponse        `json:"files,omitempty"`
}
