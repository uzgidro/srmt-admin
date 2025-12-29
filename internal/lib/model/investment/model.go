package investment

import (
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	investment_status "srmt-admin/internal/lib/model/investment-status"
	"srmt-admin/internal/lib/model/user"
	"time"
)

// Model is the internal representation of an investment
type Model struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	StatusID        int        `json:"status_id"`
	Cost            float64    `json:"cost"`
	Comments        *string    `json:"comments,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CreatedByUserID *int64     `json:"created_by_user_id,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

// ResponseModel includes joined data and files
type ResponseModel struct {
	ID            int64                   `json:"id"`
	Name          string                  `json:"name"`
	Status        investment_status.Model `json:"status"`
	Cost          float64                 `json:"cost"`
	Comments      *string                 `json:"comments,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	CreatedByUser *user.ShortInfo         `json:"created_by,omitempty"`
	UpdatedAt     *time.Time              `json:"updated_at,omitempty"`
	Files         []file.Model            `json:"files,omitempty"`
}

// ResponseWithURLs is the API response with presigned file URLs
type ResponseWithURLs struct {
	ID            int64                   `json:"id"`
	Name          string                  `json:"name"`
	Status        investment_status.Model `json:"status"`
	Cost          float64                 `json:"cost"`
	Comments      *string                 `json:"comments,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	CreatedByUser *user.ShortInfo         `json:"created_by,omitempty"`
	UpdatedAt     *time.Time              `json:"updated_at,omitempty"`
	Files         []dto.FileResponse      `json:"files,omitempty"`
}
