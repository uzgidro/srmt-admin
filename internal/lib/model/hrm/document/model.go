package document

import "time"

type HRDocument struct {
	ID           int64       `json:"id"`
	Title        string      `json:"title"`
	Type         string      `json:"type"`
	Category     string      `json:"category"`
	Number       string      `json:"number"`
	Date         string      `json:"date"`
	Status       string      `json:"status"`
	Content      *string     `json:"content,omitempty"`
	FileURL      *string     `json:"file_url,omitempty"`
	DepartmentID *int64      `json:"department_id,omitempty"`
	EmployeeID   *int64      `json:"employee_id,omitempty"`
	CreatedBy    *int64      `json:"created_by,omitempty"`
	Signatures   []Signature `json:"signatures"`
	Version      int         `json:"version"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type Signature struct {
	ID             int64      `json:"id"`
	DocumentID     int64      `json:"document_id"`
	SignerID       int64      `json:"signer_id"`
	SignerName     string     `json:"signer_name"`
	SignerPosition string     `json:"signer_position"`
	Status         string     `json:"status"`
	SignedAt       *time.Time `json:"signed_at,omitempty"`
	Comment        *string    `json:"comment,omitempty"`
	Order          int        `json:"order"`
	CreatedAt      time.Time  `json:"created_at"`
}

type DocumentRequest struct {
	ID              int64      `json:"id"`
	EmployeeID      int64      `json:"employee_id"`
	EmployeeName    string     `json:"employee_name"`
	DocumentType    string     `json:"document_type"`
	Purpose         string     `json:"purpose"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
