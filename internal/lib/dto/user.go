package dto

type GetAllUsersFilters struct {
	OrganizationID *int64
	DepartmentID   *int64
	IsActive       *bool
}

// EditUserRequest - DTO для обновления пользователя
// (Обновляет *только* таблицу users)
type EditUserRequest struct {
	Login    *string
	IsActive *bool
	RoleIDs  []int64 // Optional: if provided, replaces all user roles with this list
	// (Пароль передается отдельно)
}

// CreateUserRequest - DTO для создания пользователя
type CreateUserRequest struct {
	Login    string  `json:"login" validate:"required"`
	Password string  `json:"password" validate:"required,min=8"`
	RoleIDs  []int64 `json:"role_ids" validate:"required,min=1"`

	// XOR: Либо `ContactID`, либо `Contact`
	ContactID *int64             `json:"contact_id,omitempty" validate:"omitempty,gt=0"`
	Contact   *AddContactRequest `json:"contact,omitempty" validate:"omitempty"`
}

// UpdateUserRequest - Service layer DTO
type UpdateUserRequest struct {
	Login    *string
	Password *string
	IsActive *bool
	RoleIDs  *[]int64 // Pointer to allow "no change" vs "empty/clear" distinction if needed, or just use nil
}
