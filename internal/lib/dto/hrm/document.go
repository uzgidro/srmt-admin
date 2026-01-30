package hrm

import "time"

// --- Document Type DTOs ---

// AddDocumentTypeRequest represents request to create document type
type AddDocumentTypeRequest struct {
	Name                      string  `json:"name" validate:"required"`
	Code                      *string `json:"code,omitempty"`
	Description               *string `json:"description,omitempty"`
	TemplateID                *int64  `json:"template_id,omitempty"`
	RequiresSignature         bool    `json:"requires_signature"`
	RequiresEmployeeSignature bool    `json:"requires_employee_signature"`
	RequiresManagerSignature  bool    `json:"requires_manager_signature"`
	RequiresHRSignature       bool    `json:"requires_hr_signature"`
	ExpiryDays                *int    `json:"expiry_days,omitempty"`
	SortOrder                 int     `json:"sort_order"`
}

// EditDocumentTypeRequest represents request to update document type
type EditDocumentTypeRequest struct {
	Name                      *string `json:"name,omitempty"`
	Code                      *string `json:"code,omitempty"`
	Description               *string `json:"description,omitempty"`
	TemplateID                *int64  `json:"template_id,omitempty"`
	RequiresSignature         *bool   `json:"requires_signature,omitempty"`
	RequiresEmployeeSignature *bool   `json:"requires_employee_signature,omitempty"`
	RequiresManagerSignature  *bool   `json:"requires_manager_signature,omitempty"`
	RequiresHRSignature       *bool   `json:"requires_hr_signature,omitempty"`
	ExpiryDays                *int    `json:"expiry_days,omitempty"`
	IsActive                  *bool   `json:"is_active,omitempty"`
	SortOrder                 *int    `json:"sort_order,omitempty"`
}

// --- Document DTOs ---

// AddDocumentRequest represents request to create document
type AddDocumentRequest struct {
	EmployeeID     int64      `json:"employee_id" validate:"required"`
	DocumentTypeID int        `json:"document_type_id" validate:"required"`
	Title          string     `json:"title" validate:"required"`
	DocumentNumber *string    `json:"document_number,omitempty"`
	Description    *string    `json:"description,omitempty"`
	FileID         *int64     `json:"file_id,omitempty"`
	IssueDate      *time.Time `json:"issue_date,omitempty"`
	EffectiveDate  *time.Time `json:"effective_date,omitempty"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// EditDocumentRequest represents request to update document
type EditDocumentRequest struct {
	Title          *string    `json:"title,omitempty"`
	DocumentNumber *string    `json:"document_number,omitempty"`
	Description    *string    `json:"description,omitempty"`
	FileID         *int64     `json:"file_id,omitempty"`
	IssueDate      *time.Time `json:"issue_date,omitempty"`
	EffectiveDate  *time.Time `json:"effective_date,omitempty"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty"`
	Status         *string    `json:"status,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// DocumentFilter represents filter for documents
type DocumentFilter struct {
	EmployeeID     *int64  `json:"employee_id,omitempty"`
	DocumentTypeID *int    `json:"document_type_id,omitempty"`
	Status         *string `json:"status,omitempty"`
	ExpiringDays   *int    `json:"expiring_days,omitempty"` // Expiring within N days
	Expired        *bool   `json:"expired,omitempty"`
	Search         *string `json:"search,omitempty"` // Title, number search
	Limit          int     `json:"limit,omitempty"`
	Offset         int     `json:"offset,omitempty"`
}

// GenerateDocumentRequest represents request to generate document from template
type GenerateDocumentRequest struct {
	EmployeeID    int64                  `json:"employee_id" validate:"required"`
	TemplateID    int64                  `json:"template_id" validate:"required"`
	Title         string                 `json:"title" validate:"required"`
	Placeholders  map[string]interface{} `json:"placeholders,omitempty"`
	EffectiveDate *time.Time             `json:"effective_date,omitempty"`
	ExpiryDate    *time.Time             `json:"expiry_date,omitempty"`
}

// --- Document Signature DTOs ---

// AddSignatureRequest represents request to add signature requirement
type AddSignatureRequest struct {
	DocumentID   int64   `json:"document_id" validate:"required"`
	SignerUserID int64   `json:"signer_user_id" validate:"required"`
	SignerRole   string  `json:"signer_role" validate:"required,oneof=employee manager hr director"`
	SignOrder    int     `json:"sign_order"`
	Notes        *string `json:"notes,omitempty"`
}

// SignDocumentRequest represents request to sign document
type SignDocumentRequest struct {
	Signed bool    `json:"signed"`
	Reason *string `json:"reason,omitempty"` // If rejecting
}

// SignatureFilter represents filter for signatures
type SignatureFilter struct {
	DocumentID   *int64  `json:"document_id,omitempty"`
	SignerUserID *int64  `json:"signer_user_id,omitempty"`
	SignerRole   *string `json:"signer_role,omitempty"`
	Status       *string `json:"status,omitempty"`
}

// --- Document Template DTOs ---

// AddDocumentTemplateRequest represents request to create template
type AddDocumentTemplateRequest struct {
	DocumentTypeID int         `json:"document_type_id" validate:"required"`
	Name           string      `json:"name" validate:"required"`
	Description    *string     `json:"description,omitempty"`
	Content        *string     `json:"content,omitempty"`
	FileID         *int64      `json:"file_id,omitempty"`
	Placeholders   interface{} `json:"placeholders,omitempty"`
}

// EditDocumentTemplateRequest represents request to update template
type EditDocumentTemplateRequest struct {
	Name         *string     `json:"name,omitempty"`
	Description  *string     `json:"description,omitempty"`
	Content      *string     `json:"content,omitempty"`
	FileID       *int64      `json:"file_id,omitempty"`
	Placeholders interface{} `json:"placeholders,omitempty"`
	IsActive     *bool       `json:"is_active,omitempty"`
}

// DocumentTemplateFilter represents filter for templates
type DocumentTemplateFilter struct {
	DocumentTypeID *int    `json:"document_type_id,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
	Search         *string `json:"search,omitempty"` // Name search
}
