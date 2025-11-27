package visit

import (
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
	"time"
)

type ResponseModel struct {
	ID               int64           `json:"id"`
	OrganizationID   int64           `json:"organization_id"`
	OrganizationName string          `json:"organization_name"`
	VisitDate        time.Time       `json:"visit_date"`
	Description      string          `json:"description"`
	ResponsibleName  string          `json:"responsible_name"`
	CreatedAt        time.Time       `json:"created_at"`
	CreatedByUser    *user.ShortInfo `json:"created_by"`
	Files            []file.Model    `json:"files,omitempty"`
}
