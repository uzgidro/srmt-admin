package dto

// AddFastCallRequest - DTO for creating a fast call entry
type AddFastCallRequest struct {
	ContactID int64
	Position  int
}

// EditFastCallRequest - DTO for updating a fast call entry
type EditFastCallRequest struct {
	ContactID *int64
	Position  *int
}
