package dto

// SignDocumentRequest is the request body for signing a document
type SignDocumentRequest struct {
	ResolutionText     *string `json:"resolution_text,omitempty"`
	AssignedExecutorID *int64  `json:"assigned_executor_id,omitempty"`
	AssignedDueDate    *string `json:"assigned_due_date,omitempty"` // YYYY-MM-DD format
}

// RejectSignatureRequest is the request body for rejecting a document signature
type RejectSignatureRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// SignatureResponse is the response after signing/rejecting a document
type SignatureResponse struct {
	Status    string      `json:"status"`
	NewStatus *StatusInfo `json:"new_status,omitempty"`
}

// StatusInfo represents brief status information
type StatusInfo struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
