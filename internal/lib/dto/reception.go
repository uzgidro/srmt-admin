package dto

import (
	"time"
)

// GetAllReceptionsFilters - Filters for querying receptions
type GetAllReceptionsFilters struct {
	StartDate *time.Time // Filter receptions from this date (inclusive)
	EndDate   *time.Time // Filter receptions until this date (inclusive)
	Status    *string    // Filter by status: "default", "true", or "false"
}

// AddReceptionRequest - DTO for creating a reception
type AddReceptionRequest struct {
	Name        string
	Together    *string
	Date        time.Time
	Description *string
	Visitor     string
	CreatedByID int64
}

// EditReceptionRequest - DTO for updating a reception
type EditReceptionRequest struct {
	Name               *string
	Together           *string
	Date               *time.Time
	Description        *string
	Visitor            *string
	Status             *string // "default", "true", or "false"
	StatusChangeReason *string
	Informed           *bool
	InformedByUserID   *int64
	UpdatedByID        int64 // Required: who is making this update
}
