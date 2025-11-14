package dto

import "time"

// GetAllEventsFilters - Filters for querying events
type GetAllEventsFilters struct {
	EventStatusIDs []int      // Filter by multiple status IDs
	EventTypeIDs   []int      // Filter by multiple type IDs
	StartDate      *time.Time // Filter events from this date (inclusive)
	EndDate        *time.Time // Filter events until this date (inclusive)
	OrganizationID *int64     // Filter by organization
}

// AddEventRequest - DTO for creating an event
type AddEventRequest struct {
	Name                 string
	Description          *string
	Location             *string
	EventDate            time.Time
	ResponsibleContactID int64
	EventStatusID        int
	EventTypeID          int
	OrganizationID       *int64
	CreatedByID          int64
	FileIDs              []int64 // IDs of files to link to this event
}

// EditEventRequest - DTO for updating an event
type EditEventRequest struct {
	Name                 *string
	Description          *string
	Location             *string
	EventDate            *time.Time
	ResponsibleContactID *int64
	EventStatusID        *int
	EventTypeID          *int
	OrganizationID       *int64
	UpdatedByID          int64   // Required: who is making this update
	FileIDs              []int64 // If provided, replaces all existing file links
}
