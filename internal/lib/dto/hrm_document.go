package dto

// --- HR Documents ---

type CreateHRDocumentRequest struct {
	Title        string           `json:"title" validate:"required"`
	Type         string           `json:"type" validate:"required,oneof=order contract agreement certificate reference memo report protocol regulation instruction application other"`
	Category     string           `json:"category" validate:"required,oneof=personnel administrative financial legal other"`
	Number       string           `json:"number" validate:"required"`
	Date         string           `json:"date" validate:"required"`
	Content      *string          `json:"content,omitempty"`
	FileURL      *string          `json:"file_url,omitempty"`
	DepartmentID *int64           `json:"department_id,omitempty"`
	EmployeeID   *int64           `json:"employee_id,omitempty"`
	Signatures   []SignatureInput `json:"signatures,omitempty"`
}

type UpdateHRDocumentRequest struct {
	Title        *string `json:"title,omitempty"`
	Type         *string `json:"type,omitempty" validate:"omitempty,oneof=order contract agreement certificate reference memo report protocol regulation instruction application other"`
	Category     *string `json:"category,omitempty" validate:"omitempty,oneof=personnel administrative financial legal other"`
	Number       *string `json:"number,omitempty"`
	Date         *string `json:"date,omitempty"`
	Status       *string `json:"status,omitempty" validate:"omitempty,oneof=draft pending_review pending_signatures active archived cancelled"`
	Content      *string `json:"content,omitempty"`
	FileURL      *string `json:"file_url,omitempty"`
	DepartmentID *int64  `json:"department_id,omitempty"`
	EmployeeID   *int64  `json:"employee_id,omitempty"`
}

type HRDocumentFilters struct {
	Status   *string
	Type     *string
	Category *string
	Search   *string
}

type SignatureInput struct {
	SignerID int64 `json:"signer_id" validate:"required"`
	Order    int   `json:"order"`
}

// --- Document Requests ---

type CreateDocumentRequestReq struct {
	DocumentType string `json:"document_type" validate:"required"`
	Purpose      string `json:"purpose" validate:"required"`
}

type RejectDocumentRequestReq struct {
	Reason string `json:"reason" validate:"required"`
}
