package hrm

import "time"

// DocumentType represents HR document types
type DocumentType struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	Code *string `json:"code,omitempty"`

	Description *string `json:"description,omitempty"`

	// Template
	TemplateID *int64 `json:"template_id,omitempty"`

	// Configuration
	RequiresSignature         bool `json:"requires_signature"`
	RequiresEmployeeSignature bool `json:"requires_employee_signature"`
	RequiresManagerSignature  bool `json:"requires_manager_signature"`
	RequiresHRSignature       bool `json:"requires_hr_signature"`

	ExpiryDays *int `json:"expiry_days,omitempty"`

	IsActive  bool `json:"is_active"`
	SortOrder int  `json:"sort_order"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DocumentType code constants
const (
	DocumentTypeCodeContract     = "CONTRACT"
	DocumentTypeCodeNDA          = "NDA"
	DocumentTypeCodeJobDesc      = "JOB_DESC"
	DocumentTypeCodePolicy       = "POLICY"
	DocumentTypeCodePerfReview   = "PERF_REVIEW"
	DocumentTypeCodeTermination  = "TERMINATION"
	DocumentTypeCodePromotion    = "PROMOTION"
	DocumentTypeCodeWarning      = "WARNING"
	DocumentTypeCodeTrainingCert = "TRAINING_CERT"
	DocumentTypeCodeLeave        = "LEAVE"
)

// Document represents an HR document
type Document struct {
	ID             int64 `json:"id"`
	EmployeeID     int64 `json:"employee_id"`
	DocumentTypeID int   `json:"document_type_id"`

	// Document info
	Title          string  `json:"title"`
	DocumentNumber *string `json:"document_number,omitempty"`
	Description    *string `json:"description,omitempty"`

	// File
	FileID *int64 `json:"file_id,omitempty"`

	// Dates
	IssueDate     *time.Time `json:"issue_date,omitempty"`
	EffectiveDate *time.Time `json:"effective_date,omitempty"`
	ExpiryDate    *time.Time `json:"expiry_date,omitempty"`

	// Status
	Status string `json:"status"`

	// Created by
	CreatedBy *int64 `json:"created_by,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee     *Employee           `json:"employee,omitempty"`
	DocumentType *DocumentType       `json:"document_type,omitempty"`
	FileURL      *string             `json:"file_url,omitempty"`
	Signatures   []DocumentSignature `json:"signatures,omitempty"`
	CreatorName  *string             `json:"creator_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DocumentStatus constants
const (
	DocumentStatusDraft            = "draft"
	DocumentStatusPendingSignature = "pending_signature"
	DocumentStatusActive           = "active"
	DocumentStatusExpired          = "expired"
	DocumentStatusArchived         = "archived"
)

// DocumentSignature represents document signature
type DocumentSignature struct {
	ID         int64 `json:"id"`
	DocumentID int64 `json:"document_id"`

	// Signer
	SignerUserID int64  `json:"signer_user_id"`
	SignerRole   string `json:"signer_role"` // employee, manager, hr, director

	// Signature
	Status          string     `json:"status"`
	SignedAt        *time.Time `json:"signed_at,omitempty"`
	SignatureIP     *string    `json:"signature_ip,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`

	// Order
	SignOrder int `json:"sign_order"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	SignerName *string `json:"signer_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// SignerRole constants
const (
	SignerRoleEmployee = "employee"
	SignerRoleManager  = "manager"
	SignerRoleHR       = "hr"
	SignerRoleDirector = "director"
)

// SignatureStatus constants
const (
	SignatureStatusPending  = "pending"
	SignatureStatusSigned   = "signed"
	SignatureStatusRejected = "rejected"
)

// DocumentTemplate represents document template
type DocumentTemplate struct {
	ID             int64 `json:"id"`
	DocumentTypeID int   `json:"document_type_id"`

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	// Template content
	Content *string `json:"content,omitempty"` // HTML/Markdown with placeholders
	FileID  *int64  `json:"file_id,omitempty"` // DOCX/PDF template

	// Placeholders
	Placeholders interface{} `json:"placeholders,omitempty"` // JSON array

	// Status
	IsActive bool `json:"is_active"`
	Version  int  `json:"version"`

	CreatedBy *int64 `json:"created_by,omitempty"`

	// Enriched
	DocumentType *DocumentType `json:"document_type,omitempty"`
	FileURL      *string       `json:"file_url,omitempty"`
	CreatorName  *string       `json:"creator_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// TemplatePlaceholder represents a placeholder in template
type TemplatePlaceholder struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"` // text, date, number
}

// DocumentStats represents document metrics
type DocumentStats struct {
	TotalDocuments    int            `json:"total_documents"`
	PendingSignatures int            `json:"pending_signatures"`
	ExpiringDocuments int            `json:"expiring_documents_30d"`
	ExpiredDocuments  int            `json:"expired_documents"`
	DocumentsByType   map[string]int `json:"documents_by_type"`
}
