package personnel

import "time"

type Record struct {
	ID              int64     `json:"id"`
	EmployeeID      int64     `json:"employee_id"`
	EmployeeName    string    `json:"employee_name"`
	TabNumber       string    `json:"tab_number"`
	HireDate        string    `json:"hire_date"`
	DepartmentID    int64     `json:"department_id"`
	DepartmentName  string    `json:"department_name"`
	PositionID      int64     `json:"position_id"`
	PositionName    string    `json:"position_name"`
	ContractType    string    `json:"contract_type"`
	ContractEndDate *string   `json:"contract_end_date,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Document struct {
	ID         int64     `json:"id"`
	RecordID   int64     `json:"record_id"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	FileURL    string    `json:"file_url"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Transfer struct {
	ID             int64  `json:"id"`
	RecordID       int64  `json:"record_id"`
	FromDepartment string `json:"from_department"`
	ToDepartment   string `json:"to_department"`
	FromPosition   string `json:"from_position"`
	ToPosition     string `json:"to_position"`
	TransferDate   string `json:"transfer_date"`
	OrderNumber    string `json:"order_number"`
	Reason         string `json:"reason"`
}
