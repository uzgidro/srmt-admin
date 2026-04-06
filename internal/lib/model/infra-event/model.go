package infraevent

import (
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
	"time"
)

type ResponseModel struct {
	ID               int64           `json:"id"`
	CategoryID       int64           `json:"category_id"`
	CategorySlug     string          `json:"category_slug"`
	CategoryName     string          `json:"category_name"`
	OrganizationID   int64           `json:"organization_id"`
	OrganizationName string          `json:"organization_name"`
	OccurredAt       time.Time       `json:"occurred_at"`
	RestoredAt       *time.Time      `json:"restored_at,omitempty"`
	Description      string          `json:"description"`
	Remediation      *string         `json:"remediation,omitempty"`
	Notes            *string         `json:"notes,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	CreatedByUser    *user.ShortInfo `json:"created_by"`
	Files            []file.Model    `json:"files,omitempty"`
}

type ResponseWithURLs struct {
	ID               int64              `json:"id"`
	CategoryID       int64              `json:"category_id"`
	CategorySlug     string             `json:"category_slug"`
	CategoryName     string             `json:"category_name"`
	OrganizationID   int64              `json:"organization_id"`
	OrganizationName string             `json:"organization_name"`
	OccurredAt       time.Time          `json:"occurred_at"`
	RestoredAt       *time.Time         `json:"restored_at,omitempty"`
	Description      string             `json:"description"`
	Remediation      *string            `json:"remediation,omitempty"`
	Notes            *string            `json:"notes,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	CreatedByUser    *user.ShortInfo    `json:"created_by"`
	Files            []dto.FileResponse `json:"files,omitempty"`
}
