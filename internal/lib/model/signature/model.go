package signature

import (
	"time"

	"srmt-admin/internal/lib/model/user"
)

// ContactShort represents minimal contact information for signature
type ContactShort struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Signature represents a document signature/resolution
type Signature struct {
	ID               int64           `json:"id"`
	DocumentType     string          `json:"document_type"`
	DocumentID       int64           `json:"document_id"`
	Action           string          `json:"action"` // "signed" | "rejected"
	ResolutionText   *string         `json:"resolution_text,omitempty"`
	RejectionReason  *string         `json:"rejection_reason,omitempty"`
	AssignedExecutor *ContactShort   `json:"assigned_executor,omitempty"`
	AssignedDueDate  *time.Time      `json:"assigned_due_date,omitempty"`
	SignedBy         *user.ShortInfo `json:"signed_by,omitempty"`
	SignedAt         time.Time       `json:"signed_at"`
}

// PendingDocument represents a document waiting for signature
type PendingDocument struct {
	DocumentType    string    `json:"document_type"`
	DocumentID      int64     `json:"document_id"`
	Name            string    `json:"name"`
	Number          *string   `json:"number,omitempty"`
	DocumentDate    time.Time `json:"document_date"`
	TypeID          int       `json:"type_id"`
	TypeName        string    `json:"type_name"`
	Organization    *string   `json:"organization,omitempty"`
	OrganizationID  *int64    `json:"organization_id,omitempty"`
	ResponsibleName *string   `json:"responsible_name,omitempty"`
	ResponsibleID   *int64    `json:"responsible_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       *string   `json:"created_by,omitempty"`
}

// SignatureAction constants
const (
	ActionSigned   = "signed"
	ActionRejected = "rejected"
)

// DocumentType constants
const (
	DocTypeDecree      = "decree"
	DocTypeReport      = "report"
	DocTypeLetter      = "letter"
	DocTypeInstruction = "instruction"
)

// ValidDocumentTypes returns list of valid document types
func ValidDocumentTypes() []string {
	return []string{DocTypeDecree, DocTypeReport, DocTypeLetter, DocTypeInstruction}
}

// IsValidDocumentType checks if document type is valid
func IsValidDocumentType(docType string) bool {
	for _, t := range ValidDocumentTypes() {
		if t == docType {
			return true
		}
	}
	return false
}
